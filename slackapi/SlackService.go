package slackapi

import "github.com/nlopes/slack"

type SlackAPIService interface {
	GetUserInfo(user string) (*slack.User, error)
	GetChannelInfo(channel string) (*slack.Channel, error)
	NewRTM() *slack.RTM
}

type SlackRTMService interface {
	NewOutgoingMessage(text string, channel string) *slack.OutgoingMessage
	SendMessage(msg *slack.OutgoingMessage)
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

type SlackRTMServer struct {
	Rtm *slack.RTM
}

func (s *SlackRTMServer) NewOutgoingMessage(text string, channel string) *slack.OutgoingMessage {
	return s.Rtm.NewOutgoingMessage(text, channel)
}

func (s *SlackRTMServer) SendMessage(msg *slack.OutgoingMessage) {
	s.Rtm.SendMessage(msg)
}

func (s *SlackAPIServer) GetChannelInfo(channel string) (*slack.Channel, error) {
	return s.Api.GetChannelInfo(channel)
}
