package main

import (
	"math/rand"
	"testing"

	"github.com/Clever/kayvee-go/logger"
	"github.com/Clever/pickabot/mock_slackapi"
	whoswho "github.com/Clever/who-is-who/go-client"
	"github.com/golang/mock/gomock"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
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

func TestEmptyMessageForBot(t *testing.T) {
	pickabot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)

	pickabot.DecodeMessage(makeSlackMessage("<@U1234> "))
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
	for _, input := range []string{
		// Without "eng"
		"<@U1234> pick example-team",
		"<@U1234> pick a example-team",
		"<@U1234> pick an example-team",
		// With "eng-"
		"<@U1234> pick eng-example-team",
		"<@U1234> pick a eng-example-team",
		"<@U1234> pick an eng-example-team",
		// With "eng "
		"<@U1234> pick eng example-team",
		"<@U1234> pick a eng example-team",
		"<@U1234> pick an eng example-team",
		// With text after the team name
		"<@U1234> pick a eng-example-team for https://github.com/Clever/fake-repo/pull/1",
	} {
		t.Log("Input = ", input)
		pickabot, mocks, mockCtrl := getMockBot(t)
		defer mockCtrl.Finish()

		mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
		msg := "I choose you: <@U3>"
		message := makeSlackOutgoingMessage(msg)
		mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
		mocks.SlackRTM.EXPECT().SendMessage(message)

		pickabot.DecodeMessage(makeSlackMessage(input))
	}
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

func TestAddOverride(t *testing.T) {
	pickabot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
	msg := "Added <@U5555> to team example-team!"
	message := makeSlackOutgoingMessage(msg)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
	mocks.SlackRTM.EXPECT().SendMessage(message)

	assert.Equal(t, 0, len(teamOverrides))
	assert.Equal(t, []whoswho.User(nil), teamOverrides["example-team"])
	pickabot.DecodeMessage(makeSlackMessage("<@U1234> <@U5555> is an eng-example-team"))
	assert.Equal(t, []whoswho.User{
		whoswho.User{SlackID: "U5555"},
	}, teamOverrides["example-team"])
}
