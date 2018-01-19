package main

import (
	"fmt"
	"math/rand"
	"regexp"
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
	TeamToTeamMembers map[string][]whoswho.User
	SlackAPIService   slackapi.SlackAPIService
	SlackRTMService   slackapi.SlackRTMService
	ProjectMap        map[string]string
	Logger            logger.KayveeLogger
	Name              string
	RandomSource      rand.Source
}

// TODO: force a greedy match on user
// https://stackoverflow.com/questions/2301285/what-do-lazy-and-greedy-mean-in-the-context-of-regular-expressions
var botMessageRegex = regexp.MustCompile(`^<@(.+?)> (.*)`)
var pickTeamRegex = regexp.MustCompile(`pick a[n]? (.*)`)
var overrideTeamRegex = regexp.MustCompile(`<@(.+?)> is a[n]? (.*)`)

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

			// Pick a team member
			teamMatch := pickTeamRegex.FindStringSubmatch(message)
			if len(teamMatch) > 1 {
				teamName := teamMatch[1]
				// TODO: Return an error and handle it
				bot.pickTeamMember(ev, teamName)
				return
			}

			// Override team
			overrideMatch := overrideTeamRegex.FindStringSubmatch(message)
			if len(overrideMatch) > 2 {
				userID := overrideMatch[1]
				teamName := overrideMatch[2]
				bot.setTeamOverride(ev, userID, teamName)
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
	for team, _ := range bot.TeamToTeamMembers {
		teams = append(teams, team)
	}

	possibles := []string{}
	for _, t := range teams {
		if s == t {
			return t, nil
		}
		if lev.DistanceForStrings([]rune(s), []rune(t), lev.DefaultOptions) < 3 {
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

	bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage("It's a match! "+actualTeamName, ev.Channel))
	teamOverridesLock.Lock()
	teamOverrides[teamName] = append(teamOverrides[teamName], whoswho.User{SlackID: userID})
	teamOverridesLock.Unlock()

	fmt.Println("Team Overrides:")
	fmt.Println(teamOverrides)
	return
}

func (bot *Bot) pickTeamMember(ev *slack.MessageEvent, teamName string) {
	currentUser := whoswho.User{SlackID: ev.User}
	bot.Logger.InfoD("pick-team-member", logger.M{"team": teamName, "omit-user": currentUser.Email})

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

	userInfo, err := bot.SlackAPIService.GetUserInfo(user.SlackID)
	if err != nil {
		bot.Logger.ErrorD("get-user-info-error", logger.M{"error": err.Error(), "event-text": ev.Text})
		bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(pickUserProblem, ev.Channel))
		return
	}

	bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(fmt.Sprintf("I choose you: %s", userInfo.Name), ev.Channel))
	return
}
