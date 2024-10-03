package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/Clever/kayvee-go/logger"
	"github.com/Clever/pickabot/github"
	"github.com/Clever/pickabot/slackapi"
	"github.com/slack-go/slack"
	discovery "gopkg.in/Clever/discovery-go.v1"

	whoswho "github.com/Clever/who-is-who/go-client"
)

var lg = logger.New("pickabot")

// SlackLoop is the endless service loop pickabot remains in after startup --
// e.g. the steady-state of the bot.
func SlackLoop(s *Bot) {
	s.Logger.InfoD("start", logger.M{"message": "Starting up"})
	rtm := s.SlackAPIService.NewRTM()
	s.SlackRTMService = &slackapi.SlackRTMServer{Rtm: rtm}
	go rtm.ManageConnection()

Loop:
	for {

		// Refresh the user/team cache every 1-hour
		if time.Since(s.LastCacheRefresh) > 60*time.Minute {
			teams, overrides, userFlair, err := buildTeams(s.WhoIsWhoClient)
			if err != nil {
				s.Logger.CriticalD("user cache refresh failed. will continue without cache refresh...", logger.M{"error": err})
			} else {
				s.TeamToTeamMembers = teams
				s.TeamOverrides = overrides
				s.UserFlair = userFlair
				s.LastCacheRefresh = time.Now()
			}
		}

		select {
		case msg := <-rtm.IncomingEvents: // listening for slack events via the slack real-time messaging api
			switch ev := msg.Data.(type) { // extract the type of the slack event
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

func requireEnvVar(s string) string {
	val := os.Getenv(s)
	if val == "" {
		log.Fatalf("env var %s is not defined", s)
	}
	return val
}

func main() {

	api := slack.New(
		requireEnvVar("SLACK_ACCESS_TOKEN"),
		slack.OptionLog(log.New(os.Stdout, "pickabot: ", log.Lshortfile|log.LstdFlags)),
		slack.OptionDebug(false),
	)

	endpoint, err := discovery.URL("who-is-who", "default")
	if err != nil {
		log.Fatalf("who-is-who discovery error: %s", err)
	}
	client := whoswho.NewClient(endpoint)

	teams, overrides, userFlair, err := buildTeams(client) // populate a cached set of teams and their members
	if err != nil {
		log.Fatalf("error building teams: %s", err)
	}

	appID := requireEnvVar("GITHUB_APP_ID")
	installationID := requireEnvVar("GITHUB_INSTALLATION_ID")
	devMode := requireEnvVar("DEV_MODE") != "false"
	githubOrg := requireEnvVar("GITHUB_ORG_NAME")
	githubPrivateKey := requireEnvVar("GITHUB_PRIVATE_KEY")
	privateKeyBytes := []byte(githubPrivateKey)

	githubClient := &github.AppClient{
		AppID:          appID,
		InstallationID: installationID,
		Logger:         lg,
		PrivateKey:     privateKeyBytes,
	}
	if err != nil {
		log.Fatalf("error setting up github client: %s", err)
	}

	pickabot := &Bot{
		DevMode:           devMode,
		GithubClient:      githubClient,
		GithubOrgName:     githubOrg,
		SlackAPIService:   &slackapi.SlackAPIServer{Api: api},
		Logger:            lg,
		Name:              requireEnvVar("BOT_NAME"),
		RandomSource:      rand.NewSource(time.Now().UnixNano()),
		UserFlair:         userFlair,
		TeamOverrides:     overrides,
		TeamToTeamMembers: teams,
		WhoIsWhoClient:    client,
		LastCacheRefresh:  time.Now(),
	}

	// The below code is just prints out the teams and their members for debugging purposes and as a sanity check
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

// This method uses the who-is-who go client to populate the set of teams and their members
func buildTeams(client whoIsWhoClientIface) (map[string][]whoswho.User, []Override, map[string]string, error) {
	users, err := client.GetUserList()
	if err != nil {
		return nil, []Override{}, map[string]string{}, err
	}

	// fetch users from who-is-who
	overrides := []Override{}
	teams := map[string][]whoswho.User{}
	userFlair := map[string]string{}
	for _, u := range users {
		if !u.Active {
			continue
		}

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

		if !(strings.HasPrefix(u.Team, "Engineering - ") && u.Active) {
			continue
		}
		// Format the team name
		split := strings.Split(u.Team, " - ")
		team := split[1]
		team = strings.ToLower(team)
		team = strings.ReplaceAll(team, " ", "-")

		// Write user to teams
		teams[team] = append(teams[team], u)
	}

	return teams, overrides, userFlair, nil
}
