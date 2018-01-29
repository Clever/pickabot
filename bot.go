package main

import (
	"fmt"
	"math/rand"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/Clever/kayvee-go/logger"
	"github.com/Clever/pickabot/slackapi"
	whoswho "github.com/Clever/who-is-who/go-client"
	"github.com/nlopes/slack"
	lev "github.com/texttheater/golang-levenshtein/levenshtein"
)

// Bot is the encapsulation of the logic to respond to Slack messages, by calling out to external services
type Bot struct {
	Name   string
	Logger logger.KayveeLogger

	SlackAPIService slackapi.SlackAPIService
	SlackRTMService slackapi.SlackRTMService

	// TODO: Move all picking logic to a separate struct{}
	TeamToTeamMembers map[string][]whoswho.User
	RandomSource      rand.Source
}

const teamMatcher = `(eng)?[- ]?([a-zA-Z-]+)`

var botMessageRegex = regexp.MustCompile(`^<@(.+?)> (.*)`)
var pickTeamRegex = regexp.MustCompile(`pick\s*[a]?[n]? ` + teamMatcher)
var listTeamRegex = regexp.MustCompile(`who is\s*[a]?[n]? ` + teamMatcher)
var overrideTeamRegex = regexp.MustCompile(`<@(.+?)> is\s*[a]?[n]? ` + teamMatcher)

const didNotUnderstand = "Sorry, I didn't understand that"
const couldNotFindTeam = "Sorry, I couldn't find a team with that name"
const pickUserProblem = "Sorry, I ran into an issue picking a user. Check my logs for more details :sleuth_or_spy:"

var teamOverrides = map[string][]whoswho.User{}
var teamOverridesLock = &sync.Mutex{}

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
			bot.Logger.InfoD("listening", logger.M{"message": "Saw a message for me"})
			message := result[2]
			message = strings.Trim(message, " ")
			if message == "" {
				return
			}

			// Pick a team member
			teamMatch := pickTeamRegex.FindStringSubmatch(message)
			if len(teamMatch) > 2 {
				teamName := teamMatch[2]
				bot.pickTeamMember(ev, teamName)
				return
			}

			// Override team
			overrideMatch := overrideTeamRegex.FindStringSubmatch(message)
			if len(overrideMatch) > 3 {
				userID := overrideMatch[1]
				teamName := overrideMatch[3]
				bot.setTeamOverride(ev, userID, teamName)
				return
			}

			// List team members
			listTeamMatch := listTeamRegex.FindStringSubmatch(message)
			if len(listTeamMatch) > 2 {
				teamName := listTeamMatch[2]
				bot.listTeamMembers(ev, teamName)
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

func (bot *Bot) setTeamOverride(ev *slack.MessageEvent, userID, teamName string) {
	bot.Logger.InfoD("set-team-override", logger.M{"user": userID, "team": teamName})

	actualTeamName, err := bot.findMatchingTeam(teamName)
	if err != nil {
		bot.Logger.ErrorD("find-matching-team-error", logger.M{"error": err.Error(), "event-text": ev.Text})
		bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(couldNotFindTeam, ev.Channel))
		return
	}

	teamOverridesLock.Lock()
	defer teamOverridesLock.Unlock()
	// Ignore if it's a dup
	found := false
	for _, item := range teamOverrides[actualTeamName] {
		if item.SlackID == userID {
			found = true
		}
	}
	for _, item := range bot.TeamToTeamMembers[actualTeamName] {
		if item.SlackID == userID {
			found = true
		}
	}

	// Add to team
	if !found {
		teamOverrides[actualTeamName] = append(teamOverrides[actualTeamName], whoswho.User{SlackID: userID})
	}

	bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(fmt.Sprintf("Added <@%s> to team %s!", userID, actualTeamName), ev.Channel))
}

func (bot *Bot) pickTeamMember(ev *slack.MessageEvent, teamName string) {
	currentUser := whoswho.User{SlackID: ev.User}
	bot.Logger.InfoD("pick-team-member", logger.M{"team": teamName, "omit-user": currentUser.SlackID})

	actualTeamName, err := bot.findMatchingTeam(teamName)
	if err != nil {
		bot.Logger.ErrorD("find-matching-team-error", logger.M{"error": err.Error(), "event-text": ev.Text})
		bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(couldNotFindTeam, ev.Channel))
		return
	}

	teamMembers := append(bot.TeamToTeamMembers[actualTeamName], teamOverrides[actualTeamName]...)

	user, err := pickUser(teamMembers, &currentUser, bot.RandomSource)
	if err != nil {
		bot.Logger.ErrorD("pick-user-error", logger.M{"error": err.Error(), "event-text": ev.Text})
		bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(pickUserProblem, ev.Channel))
		return
	}

	bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(fmt.Sprintf("I choose you: <@%s>", user.SlackID), ev.Channel))
	return
}

func (bot *Bot) listTeamMembers(ev *slack.MessageEvent, teamName string) {
	bot.Logger.DebugD("list-team-members", logger.M{"team": teamName, "current-user": ev.User})
	actualTeamName, err := bot.findMatchingTeam(teamName)
	if err != nil {
		bot.Logger.ErrorD("find-matching-team-error", logger.M{"error": err.Error(), "event-text": ev.Text})
		bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(couldNotFindTeam, ev.Channel))
		return
	}

	teamMembers := append(bot.TeamToTeamMembers[actualTeamName], teamOverrides[actualTeamName]...)
	usernames := []string{}
	for _, t := range teamMembers {
		info, err := bot.SlackAPIService.GetUserInfo(t.SlackID)
		if err != nil {
			bot.Logger.ErrorD("slack-api-error", logger.M{"error": err.Error(), "event-text": ev.Text})
			return
		}
		usernames = append(usernames, info.Name)
	}
	sort.Strings(usernames)

	bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(fmt.Sprintf("Team %s has the following members: %s", actualTeamName, strings.Join(usernames, ", ")), ev.Channel))
	return
}
