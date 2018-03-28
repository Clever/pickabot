package main

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Clever/kayvee-go/logger"
	"github.com/Clever/pickabot/slackapi"
	whoswho "github.com/Clever/who-is-who/go-client"
	"github.com/google/go-github/github"
	"github.com/nlopes/slack"
	lev "github.com/texttheater/golang-levenshtein/levenshtein"
)

// Bot is the encapsulation of the logic to respond to Slack messages, by calling out to external services
type Bot struct {
	Name    string
	Logger  logger.KayveeLogger
	DevMode bool

	GithubClient      *github.Client
	GithubOrgName     string
	GithubRateLimiter *time.Ticker
	SlackAPIService   slackapi.SlackAPIService
	SlackRTMService   slackapi.SlackRTMService

	// TODO: Move all picking logic to a separate struct{}
	UserFlair         map[string]string
	TeamToTeamMembers map[string][]whoswho.User
	TeamOverrides     []Override
	RandomSource      rand.Source
}

const teamMatcher = `(eng)?[- ]?([a-zA-Z-]+)`

var botMessageRegex = regexp.MustCompile(`^<@(.+?)> (.*)`)
var pickTeamRegex = regexp.MustCompile(`^\s*pick\s*[a]?[n]?\s*` + teamMatcher)
var listTeamRegex = regexp.MustCompile(`^\s*who is\s*[a]?[n]?\s*` + teamMatcher)
var overrideTeamRegex = regexp.MustCompile(`^\s*<@(.+?)> is\s*(not)?\s*[a]?[n]? ` + teamMatcher)
var overrideTeamRegex2 = regexp.MustCompile(`^\s*(add|remove)\s+<@(.+?)>\s+(to|from)\s+` + teamMatcher)
var addFlairRegex = regexp.MustCompile(`^\s*add flair (.*)`)
var removeFlairRegex = regexp.MustCompile(`^\s*remove flair`)
var setReviewerRegex = regexp.MustCompile(`.*assign.*`)

const didNotUnderstand = "Sorry, I didn't understand that"
const couldNotFindTeam = "Sorry, I couldn't find a team with that name"
const pickUserProblem = "Sorry, I ran into an issue picking a user. Check my logs for more details :sleuth_or_spy:"

// Override denotes a team override where as user should (not) be included on a team
type Override struct {
	User    whoswho.User
	Team    string
	Include bool // true = added, false = removed
}

var teamOverridesLock = &sync.Mutex{}
var userFlairLock = &sync.Mutex{}

// DecodeMessage takes a message from the Slack loop and responds appropriately
func (bot *Bot) DecodeMessage(ev *slack.MessageEvent) {
	if ev == nil {
		return
	}

	result := botMessageRegex.FindStringSubmatch(ev.Text)
	if len(result) > 1 {
		info, err := bot.SlackAPIService.GetUserInfo(result[1])
		if err != nil {
			bot.Logger.ErrorD("listen-error", logger.M{"error": err.Error(), "event-text": ev.Text})
			return
		}

		bot.Logger.InfoD("listening", logger.M{"message": "Saw message for", "data": info.Name, "my-name": bot.Name})
		if info.Name == bot.Name {
			message := result[2]
			message = strings.Trim(message, " ")
			bot.Logger.InfoD("listening", logger.M{"message": message})
			if message == "" {
				return
			}

			// Override team
			overrideMatch := overrideTeamRegex.FindStringSubmatch(message)
			if len(overrideMatch) > 4 {
				userID := overrideMatch[1]
				addOrRemove := overrideMatch[2] != "not"
				teamName := overrideMatch[4]
				bot.setTeamOverride(ev, userID, teamName, addOrRemove)
				return
			}
			// Override team (alternate matcher)
			overrideMatch2 := overrideTeamRegex2.FindStringSubmatch(message)
			if len(overrideMatch2) > 5 {
				userID := overrideMatch2[2]
				addOrRemove := overrideMatch2[1] == "add"
				teamName := overrideMatch2[5]
				bot.setTeamOverride(ev, userID, teamName, addOrRemove)
				return
			}

			// List team members
			listTeamMatch := listTeamRegex.FindStringSubmatch(message)
			if len(listTeamMatch) > 2 {
				teamName := listTeamMatch[2]
				bot.listTeamMembers(ev, teamName)
				return

			}

			// Add flair
			addFlairMatch := addFlairRegex.FindStringSubmatch(message)
			if len(addFlairMatch) > 1 {
				flair := addFlairMatch[1]
				bot.addFlair(ev, flair)
				return
			}

			// Remove flair
			removeFlairMatch := removeFlairRegex.FindStringSubmatch(message)
			if len(removeFlairMatch) > 0 {
				bot.removeFlair(ev)
				return
			}

			// Pick a team member
			teamMatch := pickTeamRegex.FindStringSubmatch(message)
			setReviewerMatch := setReviewerRegex.FindStringSubmatch(message)
			if len(teamMatch) > 2 {
				teamName := teamMatch[2]
				bot.pickTeamMember(ev, teamName, len(setReviewerMatch) > 0)
				return
			}

			bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(didNotUnderstand, ev.Channel))
		}
	}
}

// findMatchingTeam allops smarter lookup of team name
// ex. "eng-team-name", "team-name", "team-namm" (slight misspelling)
func (bot *Bot) findMatchingTeam(s string) (string, error) {
	s = strings.TrimPrefix(s, "eng-")

	teams := []string{}
	for team := range bot.TeamToTeamMembers {
		teams = append(teams, team)
	}

	possibles := []string{}
	for _, t := range teams {
		if s == t {
			return t, nil
		}
		if lev.DistanceForStrings([]rune(s), []rune(t), lev.DefaultOptions) < 5 {
			possibles = append(possibles, t)
		}
	}

	if len(possibles) == 1 {
		return possibles[0], nil
	} else if len(possibles) > 1 {
		return "", fmt.Errorf("multiple possible matches: %s", strings.Join(possibles, ", "))
	}
	return "", fmt.Errorf("no team with that name was found")

}

func (bot *Bot) setTeamOverride(ev *slack.MessageEvent, userID, teamName string, addOrRemove bool) {
	bot.Logger.InfoD("set-team-override", logger.M{"user": userID, "team": teamName, "add-or-remove": addOrRemove})

	actualTeamName, err := bot.findMatchingTeam(teamName)
	if err != nil {
		bot.Logger.ErrorD("find-matching-team-error", logger.M{"error": err.Error(), "event-text": ev.Text})
		bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(couldNotFindTeam, ev.Channel))
		return
	}

	teamOverridesLock.Lock()
	defer teamOverridesLock.Unlock()

	// Remove user override for the current team, if already present
	foundIdx := -1
	for idx, o := range bot.TeamOverrides {
		if o.User.SlackID == userID && o.Team == actualTeamName {
			foundIdx = idx
			break
		}
	}
	if foundIdx > -1 {
		// remove current entry
		bot.TeamOverrides = append(bot.TeamOverrides[:foundIdx], bot.TeamOverrides[foundIdx+1:]...)
	}

	bot.TeamOverrides = append(bot.TeamOverrides, Override{
		User:    whoswho.User{SlackID: userID},
		Team:    actualTeamName,
		Include: addOrRemove,
	})

	if addOrRemove {
		bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(fmt.Sprintf("Added <@%s> to team %s!", userID, actualTeamName), ev.Channel))
	} else {
		bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(fmt.Sprintf("Removed <@%s> from team %s!", userID, actualTeamName), ev.Channel))
	}
}

func (bot *Bot) addFlair(ev *slack.MessageEvent, flair string) {
	bot.Logger.InfoD("add-flair", logger.M{"user": ev.User, "flair": flair})

	userFlairLock.Lock()
	defer userFlairLock.Unlock()

	bot.UserFlair[ev.User] = flair

	bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(fmt.Sprintf("<@%s>, I like your style!", ev.User), ev.Channel))
}

func (bot *Bot) removeFlair(ev *slack.MessageEvent) {
	bot.Logger.InfoD("remove-flair", logger.M{"user": ev.User})

	userFlairLock.Lock()
	defer userFlairLock.Unlock()

	delete(bot.UserFlair, ev.User)

	bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage("OK, so you don't like flair.", ev.Channel))
}

func (bot *Bot) pickTeamMember(ev *slack.MessageEvent, teamName string, setReviewer bool) {
	currentUser := whoswho.User{SlackID: ev.User}
	bot.Logger.InfoD("pick-team-member", logger.M{"team": teamName, "omit-user": currentUser.SlackID})

	actualTeamName, err := bot.findMatchingTeam(teamName)
	if err != nil {
		bot.Logger.ErrorD("find-matching-team-error", logger.M{"error": err.Error(), "event-text": ev.Text})
		bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(couldNotFindTeam, ev.Channel))
		return
	}

	teamMembers := bot.buildTeam(actualTeamName)

	user, err := pickUser(teamMembers, &currentUser, bot.RandomSource)
	if err != nil {
		bot.Logger.ErrorD("pick-user-error", logger.M{"error": err.Error(), "event-text": ev.Text})
		bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(pickUserProblem, ev.Channel))
		return
	}

	// Add flair
	flair := bot.UserFlair[user.SlackID]
	if flair != "" {
		flair = " " + flair
	}

	text := fmt.Sprintf("I choose you: <@%s>%s", user.SlackID, flair)
	bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(text, ev.Channel))
	if setReviewer {
		bot.setReviewer(ev, user)
	}
	return
}

func (bot *Bot) setReviewer(ev *slack.MessageEvent, user whoswho.User) {
	if user.Github == "" {
		bot.Logger.ErrorD("set-reviewer-error", logger.M{"error": fmt.Errorf("no valid Github account for %s", user.Email)})
		return
	}
	var reposWithReviewerSet []string
	prs := parseMessageForPRs(bot.GithubOrgName, ev.Text)
	reviewerRequest := github.ReviewersRequest{
		Reviewers: []string{user.Github},
	}
	for _, pr := range prs {
		var err error
		<-bot.GithubRateLimiter.C
		if !bot.DevMode {
			// for our dev bot we shouldn't hit the API
			_, _, err = bot.GithubClient.PullRequests.RequestReviewers(context.Background(), pr.Owner, pr.Repo, pr.PRNumber, reviewerRequest)
		}
		if err != nil {
			bot.Logger.ErrorD("set-reviewer-error", logger.M{"error": err.Error(), "event-text": ev.Text})
		} else {
			reposWithReviewerSet = append(reposWithReviewerSet, pr.Repo)
		}
	}

	if len(reposWithReviewerSet) > 0 {
		bot.Logger.InfoD("set-reviewer-success", logger.M{"repos": reposWithReviewerSet, "event-text": ev.Text})
	}

	return
}

func (bot *Bot) buildTeam(teamName string) []whoswho.User {
	teamOverridesLock.Lock()
	defer teamOverridesLock.Unlock()

	teamMembers := bot.TeamToTeamMembers[teamName]
	finalTeam := []whoswho.User{}

	// Remove some members
	for _, user := range teamMembers {
		includeUser := true
		for _, override := range bot.TeamOverrides {
			if user.SlackID == override.User.SlackID && teamName == override.Team && !override.Include {
				// user has been removed
				includeUser = false
				break
			}
		}
		if includeUser {
			finalTeam = append(finalTeam, user)
		}
	}

	// Add some members
	for _, override := range bot.TeamOverrides {
		if teamName == override.Team && override.Include {
			finalTeam = append(finalTeam, override.User)
		}
	}

	// De-dupe
	dedupedTeam := []whoswho.User{}
	for _, ft := range finalTeam {
		alreadyInTeam := false
		for _, dt := range dedupedTeam {
			if dt.SlackID == ft.SlackID {
				alreadyInTeam = true
				break
			}
		}
		if !alreadyInTeam {
			dedupedTeam = append(dedupedTeam, ft)
		}
	}

	return dedupedTeam
}

func (bot *Bot) listTeamMembers(ev *slack.MessageEvent, teamName string) {
	bot.Logger.DebugD("list-team-members", logger.M{"team": teamName, "current-user": ev.User})
	actualTeamName, err := bot.findMatchingTeam(teamName)
	if err != nil {
		bot.Logger.ErrorD("find-matching-team-error", logger.M{"error": err.Error(), "event-text": ev.Text})
		bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(couldNotFindTeam, ev.Channel))
		return
	}

	teamMembers := bot.buildTeam(actualTeamName)
	usernames := []string{}
	for _, t := range teamMembers {
		info, err := bot.SlackAPIService.GetUserInfo(t.SlackID)
		if err != nil {
			bot.Logger.ErrorD("slack-api-error", logger.M{"error": err.Error(), "event-text": ev.Text})
			return
		}

		// Add flair
		flair := bot.UserFlair[t.SlackID]
		if flair != "" {
			flair = " " + flair
		}

		usernames = append(usernames, info.Name+flair)
	}
	sort.Strings(usernames)

	bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(fmt.Sprintf("Team %s has the following members: %s", actualTeamName, strings.Join(usernames, ", ")), ev.Channel))
	return
}
