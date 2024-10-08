// These interfaces are mostly just wrappers to the slack-go package's APIs.
package slackapi

import "github.com/slack-go/slack"

type SlackAPIService interface {
	GetUserInfo(user string) (*slack.User, error)
	NewRTM() *slack.RTM
}

type SlackAPIServer struct {
	Api *slack.Client
}

func (s *SlackAPIServer) GetUserInfo(user string) (*slack.User, error) {
	return s.Api.GetUserInfo(user)
}

func (s *SlackAPIServer) NewRTM() *slack.RTM {
	return s.Api.NewRTM()
}

// SlackRTMService is an interface for the slack Real Time Messaging API.
// Specifically, this interface is used to send messages to a Slack channel.
type SlackRTMService interface {
	NewOutgoingMessage(text string, channel string) *slack.OutgoingMessage
	SendMessage(msg *slack.OutgoingMessage)
}

// Simple wrapper for the slack-go's RTM
type SlackRTMServer struct {
	Rtm *slack.RTM //RTM is a managed websocket connection for slack's real time messaging API.
}

func (s *SlackRTMServer) NewOutgoingMessage(text string, channel string) *slack.OutgoingMessage {
	return s.Rtm.NewOutgoingMessage(text, channel)
}

func (s *SlackRTMServer) SendMessage(msg *slack.OutgoingMessage) {
	s.Rtm.SendMessage(msg)
}
