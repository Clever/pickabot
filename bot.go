package main

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"github.com/Clever/kayvee-go/logger"
	"github.com/Clever/pickabot/slackapi"
	whoswho "github.com/Clever/who-is-who/go-client"
	"github.com/nlopes/slack"
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

var botMessageRegex = regexp.MustCompile(`^<@(.*)> (.*)`)
var pickTeamRegex = regexp.MustCompile(`pick a[n]? (.*)`)

const didNotUnderstand = "Sorry, I didn't understand that"
const couldNotFindTeam = "Sorry, I couldn't find a team with that name"
const pickUserProblem = "Sorry, I ran into an issue picking a user. Check my logs for more details :sleuth_or_spy:"

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
				omit := whoswho.User{SlackID: ev.User}
				bot.Logger.InfoD("pick-team-member", logger.M{"team": teamName, "omit-user": omit})

				// Get team members
				// TODO: Allow smarter lookup of team name (eng-team-name, team-name, team-name with slight misspelling, etc)
				teamName = strings.TrimPrefix(teamName, "eng-")
				teamMembers, ok := bot.TeamToTeamMembers[teamName]
				if !ok {
					bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(couldNotFindTeam, ev.Channel))
					return
				}

				user, err := pickUser(teamMembers, &omit, bot.RandomSource)
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

			bot.SlackRTMService.SendMessage(bot.SlackRTMService.NewOutgoingMessage(didNotUnderstand, ev.Channel))
		}
	}
}
