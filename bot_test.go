package main

import (
	"math/rand"
	"testing"

	"github.com/Clever/kayvee-go/logger"
	"github.com/Clever/pickabot/mock_slackapi"
	whoswho "github.com/Clever/who-is-who/go-client"
	"github.com/golang/mock/gomock"
	"github.com/nlopes/slack"
)

const testChannel = "test-channel"
const testUserID = "U0"

func makeSlackMessage(text string) *slack.MessageEvent {
	return &slack.MessageEvent{
		Msg: slack.Msg{
			User:    testUserID,
			Text:    text,
			Channel: testChannel,
		},
	}
}

func makeSlackUser(name string) *slack.User {
	return &slack.User{
		Name: name,
	}
}

func makeSlackOutgoingMessage(text string) *slack.OutgoingMessage {
	return &slack.OutgoingMessage{
		Text:    text,
		Channel: testChannel,
	}
}

type BotMocks struct {
	SlackAPI *mock_slackapi.MockSlackAPIService
	SlackRTM *mock_slackapi.MockSlackRTMService
}

func getMockBot(t *testing.T) (*Bot, *BotMocks, *gomock.Controller) {
	mockCtrl := gomock.NewController(t)
	mockSlackAPIService := mock_slackapi.NewMockSlackAPIService(mockCtrl)
	mockSlackRTMService := mock_slackapi.NewMockSlackRTMService(mockCtrl)

	pickabot := &Bot{
		SlackAPIService: mockSlackAPIService,
		SlackRTMService: mockSlackRTMService,
		TeamToTeamMembers: map[string][]whoswho.User{
			"example-team": []whoswho.User{
				whoswho.User{SlackID: "U1"},
				whoswho.User{SlackID: "U2"},
				whoswho.User{SlackID: "U3"},
				whoswho.User{SlackID: "U4"},
			},
			"empty-team": []whoswho.User{},
			"same-user-team": []whoswho.User{
				whoswho.User{SlackID: testUserID},
			},
		},
		Logger:       logger.New(testChannel),
		Name:         testUserID,
		RandomSource: rand.NewSource(0),
	}

	return pickabot, &BotMocks{
		mockSlackAPIService,
		mockSlackRTMService,
	}, mockCtrl
}

func TestMessageNotForAnyone(t *testing.T) {
	pickabot, _, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	pickabot.DecodeMessage(makeSlackMessage("hello"))
}

func TestMessageNotForBot(t *testing.T) {
	pickabot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U9876").Return(makeSlackUser("not-specbot-test"), nil)
	pickabot.DecodeMessage(makeSlackMessage("<@U9876> hey"))
}

func TestNotUnderstoodMessageForBot(t *testing.T) {
	pickabot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
	msg := didNotUnderstand
	message := makeSlackOutgoingMessage(msg)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
	mocks.SlackRTM.EXPECT().SendMessage(message)

	pickabot.DecodeMessage(makeSlackMessage("<@U1234> hey2"))
}

func TestPickTeamMember(t *testing.T) {
	pickabot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
	mocks.SlackAPI.EXPECT().GetUserInfo("U3").Return(makeSlackUser("test-user3"), nil)
	msg := "I choose you: test-user3"
	message := makeSlackOutgoingMessage(msg)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
	mocks.SlackRTM.EXPECT().SendMessage(message)

	pickabot.DecodeMessage(makeSlackMessage("<@U1234> pick a eng-example-team"))
}

func TestPickTeamMemberWorksWithoutEngPrefix(t *testing.T) {
	pickabot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
	mocks.SlackAPI.EXPECT().GetUserInfo("U3").Return(makeSlackUser("test-user3"), nil)
	msg := "I choose you: test-user3"
	message := makeSlackOutgoingMessage(msg)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
	mocks.SlackRTM.EXPECT().SendMessage(message)

	pickabot.DecodeMessage(makeSlackMessage("<@U1234> pick an example-team"))
}

func TestPickTeamMemberInvalidTeam(t *testing.T) {
	pickabot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
	msg := couldNotFindTeam
	message := makeSlackOutgoingMessage(msg)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
	mocks.SlackRTM.EXPECT().SendMessage(message)

	pickabot.DecodeMessage(makeSlackMessage("<@U1234> pick a eng-invalid-team"))
}

func TestPickUserNoUserError(t *testing.T) {
	pickabot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
	msg := pickUserProblem
	message := makeSlackOutgoingMessage(msg)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
	mocks.SlackRTM.EXPECT().SendMessage(message)

	pickabot.DecodeMessage(makeSlackMessage("<@U1234> pick a eng-empty-team"))
}

func TestPickUserNoUserErrorDueToOmit(t *testing.T) {
	pickabot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
	msg := pickUserProblem
	message := makeSlackOutgoingMessage(msg)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
	mocks.SlackRTM.EXPECT().SendMessage(message)

	pickabot.DecodeMessage(makeSlackMessage("<@U1234> pick a eng-same-user-team"))
}
