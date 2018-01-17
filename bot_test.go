package main

import (
	"math/rand"
	"testing"

	"github.com/Clever/kayvee-go/logger"
	"github.com/Clever/pickabot/mock_slackapi"
	"github.com/golang/mock/gomock"
	"github.com/nlopes/slack"
)

const testChannel = "test-channel"
const testUser = "test-user"

func makeSlackMessage(text string) *slack.MessageEvent {
	return &slack.MessageEvent{
		Msg: slack.Msg{
			Username: testUser,
			Text:     text,
			Channel:  testChannel,
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
		TeamToTeamMembers: map[string][]User{
			"eng-example-team": []User{
				User{SlackHandle: "test-user1"},
				User{SlackHandle: "test-user2"},
				User{SlackHandle: "test-user3"},
				User{SlackHandle: "test-user4"},
			},
			"eng-empty-team": []User{},
			"eng-same-user-team": []User{
				User{SlackHandle: testUser},
			},
		},
		RepoToShepherds: map[string][]User{
			"repo1": []User{},
		},
		Logger:       logger.New(testChannel),
		Name:         testUser,
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

	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUser), nil)
	msg := didNotUnderstand
	message := makeSlackOutgoingMessage(msg)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
	mocks.SlackRTM.EXPECT().SendMessage(message)

	pickabot.DecodeMessage(makeSlackMessage("<@U1234> hey2"))
}

func TestPickTeamMember(t *testing.T) {
	pickabot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUser), nil)
	msg := "I choose you: test-user3"
	message := makeSlackOutgoingMessage(msg)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
	mocks.SlackRTM.EXPECT().SendMessage(message)

	pickabot.DecodeMessage(makeSlackMessage("<@U1234> pick a eng-example-team"))
}

func TestPickTeamMemberInvalidTeam(t *testing.T) {
	pickabot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUser), nil)
	msg := couldNotFindTeam
	message := makeSlackOutgoingMessage(msg)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
	mocks.SlackRTM.EXPECT().SendMessage(message)

	pickabot.DecodeMessage(makeSlackMessage("<@U1234> pick a eng-invalid-team"))
}

func TestPickUserNoUserError(t *testing.T) {
	pickabot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUser), nil)
	msg := pickUserProblem
	message := makeSlackOutgoingMessage(msg)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
	mocks.SlackRTM.EXPECT().SendMessage(message)

	pickabot.DecodeMessage(makeSlackMessage("<@U1234> pick a eng-empty-team"))
}

func TestPickUserNoUserErrorDueToOmit(t *testing.T) {
	pickabot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUser), nil)
	msg := pickUserProblem
	message := makeSlackOutgoingMessage(msg)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
	mocks.SlackRTM.EXPECT().SendMessage(message)

	pickabot.DecodeMessage(makeSlackMessage("<@U1234> pick a eng-same-user-team"))
}
