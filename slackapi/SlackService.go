// These interfaces are mostly just wrappers to the slack-go package's APIs.
package slackapi

import (
	"fmt"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

type SlackAPIService interface {
	GetUserInfo(user string) (*slack.User, error)
	GetAPI() *slack.Client
}

type SlackAPIServer struct {
	Api *slack.Client
}

func (s *SlackAPIServer) GetAPI() *slack.Client {
	return s.Api
}

func (s *SlackAPIServer) GetUserInfo(user string) (*slack.User, error) {
	return s.Api.GetUserInfo(user)
}

// SlackEventsService is an interface for the Slack Socket Mode API
// Used to send messages to Slack channels
type SlackEventsService interface {
	PostMessage(channel string, text string) error
}

// SlackEventsClient wraps the socketmode client
type SlackEventsClient struct {
	Client *socketmode.Client
}

func (s *SlackEventsClient) PostMessage(channel string, text string) error {
	_, _, err := s.Client.PostMessage(channel, slack.MsgOptionText(text, false))
	if err != nil {
		return fmt.Errorf("failed to send message: %s", err)
	}
	return nil
}
