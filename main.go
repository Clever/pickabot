package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/Clever/kayvee-go/logger"
	"github.com/Clever/pickabot/slackapi"
	"github.com/nlopes/slack"

	"github.com/Clever/discovery-go"
	whoswho "github.com/Clever/who-is-who/go-client"
)

// SlackLoop is the main Slack loop for specbot, to listen for commands
func SlackLoop(s *Bot) {
	s.Logger.InfoD("start", logger.M{"message": "Starting up"})
	rtm := s.SlackAPIService.NewRTM()
	s.SlackRTMService = &slackapi.SlackRTMServer{Rtm: rtm}
	go rtm.ManageConnection()

Loop:
	for {
		select {
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.MessageEvent:
				s.DecodeMessage(ev)

			case *slack.RTMError:
				s.Logger.CriticalD("listening", logger.M{"error": ev.Error()})

			case *slack.ConnectionErrorEvent:
				s.Logger.CriticalD("listening", logger.M{"error": ev.Error()})

			case *slack.InvalidAuthEvent:
				s.Logger.CriticalD("listening", logger.M{"error": "invalid credentials"})
				break Loop

			default:
				// Ignore other events..
			}
		}
	}
}

func main() {

	api := slack.New(os.Getenv("SLACK_ACCESS_TOKEN"))
	slack.SetLogger(log.New(os.Stdout, "specbot: ", log.Lshortfile|log.LstdFlags))
	api.SetDebug(false)

	teams, overrides, err := buildTeams()
	if err != nil {
		log.Fatalf("error building teams: %s", err)
	}

	pickabot := &Bot{
		SlackAPIService:   &slackapi.SlackAPIServer{Api: api},
		Logger:            logger.New("pickabot"),
		Name:              os.Getenv("BOT_NAME"),
		RandomSource:      rand.NewSource(time.Now().UnixNano()),
		TeamOverrides:     overrides,
		TeamToTeamMembers: teams,
	}

	for teamName := range teams {
		fmt.Printf("team=%s has members: ", teamName)
		users := pickabot.buildTeam(teamName)
		userText := []string{}
		for _, u := range users {
			userText = append(userText, fmt.Sprintf("%s %s (%s)", u.FirstName, u.LastName, u.SlackID))
		}
		fmt.Println(strings.Join(userText, ", "))
	}

	SlackLoop(pickabot)
}

func buildTeams() (map[string][]whoswho.User, []Override, error) {
	endpoint, err := discovery.URL("who-is-who", "default")
	if err != nil {
		return nil, []Override{}, fmt.Errorf("discovery error: %s", err)
	}

	client := whoswho.NewClient(endpoint)
	users, err := client.GetUserList()
	if err != nil {
		return nil, []Override{}, err
	}

	// fetch users from who-is-who
	overrides := []Override{}
	teams := map[string][]whoswho.User{}
	for _, u := range users {
		// Add overrides from who-is-who
		for _, to := range u.Pickabot.TeamOverrides {
			overrides = append(overrides, Override{
				User:        u,
				Team:        to.Team,
				AddOrRemove: to.AddOrRemove == "add",
			})
		}

		if !(strings.HasPrefix(u.Team, "Engineer") && u.Active) {
			continue
		}
		// Format the team name
		split := strings.Split(u.Team, " - ")
		team := split[1]
		team = strings.ToLower(team)
		team = strings.Replace(team, " ", "-", 1)

		// Write user to teams
		teams[team] = append(teams[team], u)
	}

	return teams, overrides, nil
}
