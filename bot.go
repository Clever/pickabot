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
	"github.com/Clever/pickabot/github"
	"github.com/Clever/pickabot/oncall"
	"github.com/Clever/pickabot/slackapi"
	whoswho "github.com/Clever/who-is-who/go-client"
	pagerduty "github.com/PagerDuty/go-pagerduty"
	"github.com/nlopes/slack"
	lev "github.com/texttheater/golang-levenshtein/levenshtein"
)

// generate mocks of dependencies for use during testing
//go:generate sh -c "$PWD/bin/mockgen -package main -source $PWD/slackapi/SlackService.go SlackAPIService,SlackRTMService > slack_service_mock_test.go"
//go:generate sh -c "$PWD/bin/mockgen -package main -source $PWD/github/client.go AppClientIface > github_mock_test.go"
//go:generate sh -c "$PWD/bin/mockgen -package main -source $PWD/oncall/pagerduty.go PagerDutyClientInterface > pagerduty_client_mock_test.go"

// Bot is the encapsulation of the logic to respond to Slack messages, by calling out to external services
type Bot struct {
	Name    string
	Logger  logger.KayveeLogger
	DevMode bool

	GithubClient    github.AppClientIface
	GithubOrgName   string
	PagerDutyClient oncall.PagerDutyClientInterface
	SlackAPIService slackapi.SlackAPIService
	SlackRTMService slackapi.SlackRTMService

	// TODO: Move all picking logic to a separate struct{}
	UserFlair         map[string]string
	TeamToTeamMembers map[string][]whoswho.User
	TeamOverrides     []Override
	RandomSource      rand.Source
	WhoIsWhoClient    whoIsWhoClientIface
}

const teamMatcher = `#?(eng)?[- ]?([a-zA-Z-]+)`
const individualMatcher = `<@([a-zA-Z0-9-]+)>`

var botMessageRegex = regexp.MustCompile(`^<@(.+?)> (.*)`)
var pickTeamRegex = regexp.MustCompile(`^\s*(pick\ and\ assign|pick|assign)\s*[a]?[n]?\s*` + teamMatcher)
var pickIndividualRegex = regexp.MustCompile(`^\s*(pick\ and\ assign|pick|assign)\s*[a]?[n]?\s*` + individualMatcher)
var listTeamRegex = regexp.MustCompile(`^\s*who is\s*[ai]?[n]?\s*` + teamMatcher)
var overrideTeamRegex = regexp.MustCompile(`^\s*<@(.+?)> is\s*(not)?\s*[a]?[n]? ` + teamMatcher)
var overrideTeamRegex2 = regexp.MustCompile(`^\s*(add|remove)\s+<@(.+?)>\s+(to|from)\s+` + teamMatcher)
var addFlairRegex = regexp.MustCompile(`^\s*add flair (.*)`)
var removeFlairRegex = regexp.MustCompile(`^\s*remove flair`)
var setAssigneeRegex = regexp.MustCompile(`.*assign.*`)
var helpRegex = regexp.MustCompile(`^\s*help`)

const didNotUnderstand = "Sorry, I didn't understand that"
const couldNotFindTeam = "Sorry, I couldn't find a team with that name"
const pickUserProblem = "Sorry, I ran into an issue picking a user. Check my logs for more details :sleuth_or_spy:"
const helpMessage = "_Pika-pi!_\n\nI can do the following:\n\n" +
	"`@pickabot pick a <team>` - picks a user from that team\n" +
	"`@pickabot assign a <team> for <Github PR URL(s)>` - assigns a user from that team to the Github PR(s)\n" +
	"`@pickabot who is <team>` - lists users who belong to that team\n" +
	"`@pickabot add @user to <team>` - adds user to team\n" +
	"`@pickabot remove @user from <team>` - removes user from team\n" +
	"`@pickabot add flair :emoji:` - set flair that appears when you're picked\n" +
	"`@pickabot remove flair` - remove your flair"

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

			// Help
			helpMatch := helpRegex.FindStringSubmatch(message)
			if len(helpMatch) > 0 {
				bot.Logger.Info("help match")
				// TODO: Print help
				bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(helpMessage, ev.Channel))
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

			// Determine if doing PR assignment
			setAssigneeMatch := setAssigneeRegex.FindStringSubmatch(message)
			setAssignee := len(setAssigneeMatch) > 0

			// Check if picking an individual
			// TODO: must come before team because team regex also matches individual regex
			individualMatch := pickIndividualRegex.FindStringSubmatch(message)
			if len(individualMatch) > 2 {
				individualName := individualMatch[2]
				bot.pickIndividual(ev, individualName, setAssignee)
				return
			}

			// Pick a team member
			teamMatch := pickTeamRegex.FindStringSubmatch(message)
			if len(teamMatch) > 3 {
				teamName := teamMatch[3]
				bot.pickTeamMember(ev, teamName, setAssignee)
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
		if lev.DistanceForStrings([]rune(s), []rune(t), lev.DefaultOptions) < 2 {
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

func (bot *Bot) setTeamOverrideInWhoIsWho(slackID, team string, include bool, until time.Time) {
	// If user is in WIW update it,
	user, err := bot.WhoIsWhoClient.UserBySlackID(slackID)
	if err != nil {
		bot.Logger.ErrorD("set-team-override-wiw-user-by-slack", logger.M{"user": slackID, "error": err.Error()})
		return
	}

	o := whoswho.PickabotTeamOverride{
		Team:    team,
		Include: include,
		Until:   until.Unix(),
	}

	// Remove any existing override for current team
	foundIdx := -1
	for idx, override := range user.Pickabot.TeamOverrides {
		if override.Team == team {
			foundIdx = idx
		}
	}
	if foundIdx > -1 {
		// remove current entry
		user.Pickabot.TeamOverrides = append(user.Pickabot.TeamOverrides[:foundIdx], user.Pickabot.TeamOverrides[foundIdx+1:]...)
	}

	// Add override
	user.Pickabot.TeamOverrides = append(user.Pickabot.TeamOverrides, o)

	_, err = bot.WhoIsWhoClient.UpsertUser("pickabot", user)
	if err != nil {
		bot.Logger.ErrorD("set-team-override-wiw-upsert-user", logger.M{"user": slackID, "error": err.Error()})
		return
	}
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

	bot.setTeamOverrideInWhoIsWho(userID, actualTeamName, addOrRemove, time.Time{})

	if addOrRemove {
		bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(
			fmt.Sprintf("Added <@%s> to team %s! Remember to update https://github.com/orgs/Clever/teams/eng-%s/edit/review_assignment too!", userID, actualTeamName, actualTeamName),
			ev.Channel))
	} else {
		bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(
			fmt.Sprintf("Removed <@%s> from team %s! Remember to update https://github.com/orgs/Clever/teams/eng-%s/edit/review_assignment too!", userID, actualTeamName, actualTeamName),
			ev.Channel))
	}
}

func (bot *Bot) updateFlairInWhoIsWho(slackID, flair string) {
	// If user is in WIW update it,
	user, err := bot.WhoIsWhoClient.UserBySlackID(slackID)
	if err != nil {
		bot.Logger.ErrorD("add-flair-wiw-user-by-slack", logger.M{"user": slackID, "flair": flair, "error": err.Error()})
		return
	}

	user.Pickabot.Flair = flair
	_, err = bot.WhoIsWhoClient.UpsertUser("pickabot", user)
	if err != nil {
		bot.Logger.ErrorD("add-flair-wiw-upsert-user", logger.M{"user": slackID, "flair": flair, "error": err.Error()})
		return
	}
}

func (bot *Bot) addFlair(ev *slack.MessageEvent, flair string) {
	bot.Logger.InfoD("add-flair", logger.M{"user": ev.User, "flair": flair})

	userFlairLock.Lock()
	defer userFlairLock.Unlock()

	bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(fmt.Sprintf("<@%s>, I like your style!", ev.User), ev.Channel))

	bot.UserFlair[ev.User] = flair
	bot.updateFlairInWhoIsWho(ev.User, flair)
}

func (bot *Bot) removeFlair(ev *slack.MessageEvent) {
	bot.Logger.InfoD("remove-flair", logger.M{"user": ev.User})

	userFlairLock.Lock()
	defer userFlairLock.Unlock()

	bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage("OK, so you don't like flair.", ev.Channel))

	delete(bot.UserFlair, ev.User)
	bot.updateFlairInWhoIsWho(ev.User, "")
}

func (bot *Bot) pickTeamMember(ev *slack.MessageEvent, teamName string, setAssignee bool) {
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
	if setAssignee {
		err = bot.setAssignee(ev, user)
		if err != nil {
			text = fmt.Sprintf("Error setting <@%s>%s as pull-request reviewer: %s", user.SlackID, flair, err.Error())
		} else {
			text = fmt.Sprintf("Set <@%s>%s as pull-request reviewer", user.SlackID, flair)
		}
	}
	bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(text, ev.Channel))
	return
}

func (bot *Bot) pickIndividual(ev *slack.MessageEvent, individualSlackID string, setAssignee bool) {
	bot.Logger.InfoD("pick-individual", logger.M{"slack ID": individualSlackID})
	user, err := bot.WhoIsWhoClient.UserBySlackID(individualSlackID)
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
	if setAssignee {
		err = bot.setAssignee(ev, user)
		if err != nil {
			text = fmt.Sprintf("Error setting <@%s>%s as pull-request reviewer: %s", user.SlackID, flair, err.Error())
		} else {
			text = fmt.Sprintf("Set <@%s>%s as pull-request reviewer", user.SlackID, flair)
		}
	}
	bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(text, ev.Channel))
	return
}

func (bot *Bot) setAssignee(ev *slack.MessageEvent, user whoswho.User) error {
	var err error
	if user.Github == "" {
		// try to fetch the user from SlackID
		user, err = bot.WhoIsWhoClient.UserBySlackID(user.SlackID)
		// error if there is still no valid account associated
		if user.Github == "" {
			bot.Logger.ErrorD("set-assignee-error", logger.M{
				"error":                fmt.Sprintf("no valid Github account for %s", user.Email),
				"event-text":           ev.Text,
				"user-pickabot-config": user.Pickabot,
				"user-slack":           user.Slack,
				"user-slack-id":        user.SlackID,
			})
			return fmt.Errorf("no github account for slack user <@%s>", user.SlackID)
		}
		// bubble up who-is-who-error
		if err != nil {
			bot.Logger.ErrorD("set-assignee-wiw-error", logger.M{
				"error":      err.Error(),
				"event-text": ev.Text,
			})
			return fmt.Errorf("error fetching <@%s> from who-is-who. Please manually assign instead", user.SlackID)
		}
	}
	var reposWithAssigneeSet []string
	var reposWithReviewerSet []string
	prs := parseMessageForPRs(bot.GithubOrgName, ev.Text)
	for _, pr := range prs {
		var err error
		// the dev bot shouldn't hit the API
		if bot.DevMode {
			bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(fmt.Sprintf("would have assigned %s to %s", user.Github, pr.Repo), ev.Channel))
		} else {
			_, _, err = bot.GithubClient.AddAssignees(context.Background(), pr.Owner, pr.Repo, pr.PRNumber, []string{user.Github})
			if err != nil {
				bot.Logger.ErrorD("set-assignee-failure-warning", logger.M{"warning": err.Error(), "event-text": ev.Text, "repo": pr.Repo, "user": user.Github})
			} else {
				reposWithAssigneeSet = append(reposWithAssigneeSet, pr.Repo)
			}
			_, _, err = bot.GithubClient.AddReviewers(context.Background(), pr.Owner, pr.Repo, pr.PRNumber, []string{user.Github})
			if err != nil {
				bot.Logger.ErrorD("set-reviewer-failure-warning", logger.M{"warning": err.Error(), "event-text": ev.Text, "repo": pr.Repo, "user": user.Github})
			} else {
				reposWithReviewerSet = append(reposWithReviewerSet, pr.Repo)
			}
		}
	}

	if len(reposWithAssigneeSet) > 0 || len(reposWithReviewerSet) > 0 {
		bot.Logger.InfoD("set-assignee-success", logger.M{
			"assigned-repos":  reposWithAssigneeSet,
			"reviewing-repos": reposWithReviewerSet,
			"event-text":      ev.Text,
			"user":            user.Github,
		})
	}

	return nil
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
