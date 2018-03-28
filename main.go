package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/Clever/kayvee-go/logger"
	"github.com/Clever/pickabot/slackapi"
	"github.com/google/go-github/github"
	"github.com/nlopes/slack"
	"golang.org/x/oauth2"

	"github.com/Clever/discovery-go"
	whoswho "github.com/Clever/who-is-who/go-client"
)

// Github's rate limit for authenticated requests is 5000 QPH = 83.3 QPM = 1.38 QPS = 720ms/query
// We also use a global limiter to prevent concurrent requests, which trigger Github's abuse detection
var githubLimiter = time.NewTicker(720 * time.Millisecond)

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

	teams, overrides, userFlair, err := buildTeams()
	if err != nil {
		log.Fatalf("error building teams: %s", err)
	}

	if os.Getenv("GITHUB_API_TOKEN") == "" {
		log.Fatalf("GITHUB_API_TOKEN env var is not set. In order to use pickabot, create a token (https://help.github.com/articles/creating-a-personal-access-token-for-the-command-line/) then set the env var.")
	}
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv("GITHUB_API_TOKEN")},
	)
	tc := oauth2.NewClient(ctx, ts)
	githubClient := github.NewClient(tc)
	devMode := os.Getenv("DEV_MODE") != "false"
	githubOrg := os.Getenv("GITHUB_ORG_NAME")

	pickabot := &Bot{
		DevMode:           devMode,
		GithubClient:      githubClient,
		GithubOrgName:     githubOrg,
		GithubRateLimiter: githubLimiter,
		SlackAPIService:   &slackapi.SlackAPIServer{Api: api},
		Logger:            logger.New("pickabot"),
		Name:              os.Getenv("BOT_NAME"),
		RandomSource:      rand.NewSource(time.Now().UnixNano()),
		UserFlair:         userFlair,
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

func buildTeams() (map[string][]whoswho.User, []Override, map[string]string, error) {
	endpoint, err := discovery.URL("who-is-who", "default")
	if err != nil {
		return nil, []Override{}, map[string]string{}, fmt.Errorf("discovery error: %s", err)
	}

	client := whoswho.NewClient(endpoint)
	users, err := client.GetUserList()
	if err != nil {
		return nil, []Override{}, map[string]string{}, err
	}

	// fetch users from who-is-who
	overrides := []Override{}
	teams := map[string][]whoswho.User{}
	userFlair := map[string]string{}
	for _, u := range users {
		// Add overrides from who-is-who
		for _, to := range u.Pickabot.TeamOverrides {
			overrides = append(overrides, Override{
				User:    u,
				Team:    to.Team,
				Include: to.Include,
			})
		}

		// Add flair
		if u.Pickabot.Flair != "" {
			userFlair[u.SlackID] = u.Pickabot.Flair
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

	return teams, overrides, userFlair, nil
}
