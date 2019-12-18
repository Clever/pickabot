package main

import (
	"math/rand"
	"testing"

	"github.com/Clever/kayvee-go/logger"
	whoswho "github.com/Clever/who-is-who/go-client"
	"github.com/golang/mock/gomock"
	"github.com/nlopes/slack"
	"github.com/stretchr/testify/assert"
)

const testChannel = "test-channel"
const testUserID = "U0"
const testGithubOrg = "Clever"

var testGithubUser = whoswho.User{SlackID: "G1", Github: "github"}

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
	SlackAPI       *MockSlackAPIService
	SlackRTM       *MockSlackRTMService
	WhoIsWhoClient *MockwhoIsWhoClientIface
	GithubClient   *MockAppClientIface
}

func getMockBot(t *testing.T) (*Bot, *BotMocks, *gomock.Controller) {
	mockCtrl := gomock.NewController(t)
	mockSlackAPIService := NewMockSlackAPIService(mockCtrl)
	mockSlackRTMService := NewMockSlackRTMService(mockCtrl)
	mockWhoIsWhoClient := NewMockwhoIsWhoClientIface(mockCtrl)
	mockGithubClient := NewMockAppClientIface(mockCtrl)

	mockbot := &Bot{
		SlackAPIService: mockSlackAPIService,
		SlackRTMService: mockSlackRTMService,
		UserFlair:       map[string]string{},
		TeamOverrides:   []Override{},
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
			"github-user-team": []whoswho.User{testGithubUser, whoswho.User{SlackID: "G2", Github: "G2Github"}},
		},
		Logger:         logger.New(testChannel),
		Name:           testUserID,
		RandomSource:   rand.NewSource(0),
		WhoIsWhoClient: mockWhoIsWhoClient,
		GithubClient:   mockGithubClient,
		GithubOrgName:  testGithubOrg,
	}

	return mockbot, &BotMocks{
		SlackAPI:       mockSlackAPIService,
		SlackRTM:       mockSlackRTMService,
		WhoIsWhoClient: mockWhoIsWhoClient,
		GithubClient:   mockGithubClient,
	}, mockCtrl
}

func TestMessageNotForAnyone(t *testing.T) {
	mockbot, _, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mockbot.DecodeMessage(makeSlackMessage("hello"))
}

func TestMessageNotForBot(t *testing.T) {
	mockbot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U9876").Return(makeSlackUser("not-mockbot-test"), nil)
	mockbot.DecodeMessage(makeSlackMessage("<@U9876> hey"))
}

func TestEmptyMessageForBot(t *testing.T) {
	mockbot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)

	mockbot.DecodeMessage(makeSlackMessage("<@U1234> "))
}

func TestNotUnderstoodMessageForBot(t *testing.T) {
	mockbot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
	msg := didNotUnderstand
	message := makeSlackOutgoingMessage(msg)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
	mocks.SlackRTM.EXPECT().SendMessage(message)

	mockbot.DecodeMessage(makeSlackMessage("<@U1234> hey2"))
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
		// With "#"
		"<@U1234> pick an #eng-example-team",
		// With text after the team name
		"<@U1234> pick a eng-example-team for https://github.com/Clever/fake-repo/pull/1",
	} {
		t.Log("Input = ", input)
		mockbot, mocks, mockCtrl := getMockBot(t)
		defer mockCtrl.Finish()

		mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
		msg := "I choose you: <@U3>"
		message := makeSlackOutgoingMessage(msg)
		mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
		mocks.SlackRTM.EXPECT().SendMessage(message)

		mockbot.DecodeMessage(makeSlackMessage(input))
	}
}

func TestPickAssignTeamMember(t *testing.T) {
	for _, input := range []string{
		// Without "eng"
		"<@U1234> assign a github-user-team",
		"<@U1234> assign an github-user-team",
		"<@U1234> pick and assign an github-user-team",
		"<@U1234> pick and assign a github-user-team",
		// With "eng-"
		"<@U1234> pick and assign eng-github-user-team",
		"<@U1234> pick and assign a eng-github-user-team",
		"<@U1234> pick and assign an eng-github-user-team",
		"<@U1234> assign eng-github-user-team",
		"<@U1234> assign a eng-github-user-team",
		"<@U1234> assign an eng-github-user-team",
		// With "eng "
		"<@U1234> pick and assign eng github-user-team",
		"<@U1234> pick and assign a eng github-user-team",
		"<@U1234> pick and assign an eng github-user-team",
		"<@U1234> assign eng github-user-team",
		"<@U1234> assign a eng github-user-team",
		"<@U1234> assign an eng github-user-team",
		// With "#"
		"<@U1234> pick and assign an #eng-github-user-team",
		"<@U1234> assign an #eng-github-user-team",
	} {
		t.Log("Input = ", input)
		mockbot, mocks, mockCtrl := getMockBot(t)
		defer mockCtrl.Finish()

		mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
		msg := "Set <@G1> as pull-request reviewer"
		message := makeSlackOutgoingMessage(msg)
		mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
		mocks.SlackRTM.EXPECT().SendMessage(message)

		mockbot.DecodeMessage(makeSlackMessage(input))
	}
}

func TestAssignTeamMember(t *testing.T) {
	for _, test := range []struct {
		name            string
		inputMessage    string
		expectedUser    string
		expectations    func(*BotMocks)
		expectedMessage string
	}{
		{
			name:         "no calls if the user doesn't have a github account",
			inputMessage: "<@U1234> assign a example-team for https://github.com/Clever/fake-repo/pull/1",
			expectedUser: "U3",
			expectations: func(mocks *BotMocks) {
				mocks.WhoIsWhoClient.EXPECT().UserBySlackID(gomock.Any()).Return(whoswho.User{SlackID: "U3"}, nil)
			},
			expectedMessage: "Error setting <@U3> as pull-request reviewer: no github account for slack user <@U3>",
		},
		{
			name:         "calls assign and review for a team member",
			inputMessage: "<@U1234> assign a github-user-team for https://github.com/Clever/fake-repo/pull/1 https://github.com/Clever/fake-repo2/pull/1",
			expectedUser: testGithubUser.SlackID,
			expectations: func(mocks *BotMocks) {
				// check calls
				mocks.GithubClient.EXPECT().AddAssignees(gomock.Any(), testGithubOrg, "fake-repo", 1, gomock.Any())
				mocks.GithubClient.EXPECT().AddReviewers(gomock.Any(), testGithubOrg, "fake-repo", 1, gomock.Any())
				mocks.GithubClient.EXPECT().AddAssignees(gomock.Any(), testGithubOrg, "fake-repo2", 1, gomock.Any())
				mocks.GithubClient.EXPECT().AddReviewers(gomock.Any(), testGithubOrg, "fake-repo2", 1, gomock.Any())
			},
			expectedMessage: "Set <@G1> as pull-request reviewer",
		},
	} {
		t.Logf("Case: %s. Input: %s", test.name, test.inputMessage)
		mockbot, mocks, mockCtrl := getMockBot(t)
		defer mockCtrl.Finish()

		mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
		test.expectations(mocks)
		message := makeSlackOutgoingMessage(test.expectedMessage)
		mocks.SlackRTM.EXPECT().NewOutgoingMessage(test.expectedMessage, testChannel).Return(message)
		mocks.SlackRTM.EXPECT().SendMessage(message)

		mockbot.DecodeMessage(makeSlackMessage(test.inputMessage))
	}
}

func TestPickTeamMemberInvalidTeam(t *testing.T) {
	mockbot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
	msg := couldNotFindTeam
	message := makeSlackOutgoingMessage(msg)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
	mocks.SlackRTM.EXPECT().SendMessage(message)

	mockbot.DecodeMessage(makeSlackMessage("<@U1234> pick a eng-invalid-team"))
}

func TestPickIndividual(t *testing.T) {
	for _, input := range []string{
		"<@U1234> pick <@U5>",
		"<@U1234> pick a <@U5>",
		"<@U1234> pick an <@U5>",
		"<@U1234> pick <@U5> for https://github.com/Clever/fake-repo/pull/1",
	} {
		t.Log("Input = ", input)
		mockbot, mocks, mockCtrl := getMockBot(t)
		defer mockCtrl.Finish()

		mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
		mocks.WhoIsWhoClient.EXPECT().UserBySlackID("U5").Return(whoswho.User{SlackID: "U5"}, nil)
		msg := "I choose you: <@U5>"
		message := makeSlackOutgoingMessage(msg)
		mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
		mocks.SlackRTM.EXPECT().SendMessage(message)

		mockbot.DecodeMessage(makeSlackMessage(input))
	}
}

func TestPickAssignIndividual(t *testing.T) {
	for _, input := range []string{
		"<@U1234> assign <@G1>",
		"<@U1234> assign a <@G1>",
		"<@U1234> assign an <@G1>",
		"<@U1234> pick and assign <@G1>",
		"<@U1234> pick and assign a <@G1>",
		"<@U1234> pick and assign an <@G1>",
	} {
		t.Log("Input = ", input)
		mockbot, mocks, mockCtrl := getMockBot(t)
		defer mockCtrl.Finish()

		mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
		mocks.WhoIsWhoClient.EXPECT().UserBySlackID("G1").Return(testGithubUser, nil)
		msg := "Set <@G1> as pull-request reviewer"
		message := makeSlackOutgoingMessage(msg)
		mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
		mocks.SlackRTM.EXPECT().SendMessage(message)

		mockbot.DecodeMessage(makeSlackMessage(input))
	}
}

func TestAssignIndividual(t *testing.T) {
	for _, test := range []struct {
		name            string
		inputMessage    string
		expectedUser    string
		expectations    func(*BotMocks)
		expectedMessage string
	}{
		{
			name:         "no github calls if the user doesn't have a github account",
			inputMessage: "<@U1234> assign <@U5> for https://github.com/Clever/fake-repo/pull/1",
			expectedUser: "U5",
			expectations: func(mocks *BotMocks) {
				mocks.WhoIsWhoClient.EXPECT().UserBySlackID("U5").Return(whoswho.User{SlackID: "U5"}, nil)
				mocks.WhoIsWhoClient.EXPECT().UserBySlackID("U5").Return(whoswho.User{SlackID: "U5"}, nil)
			},
			expectedMessage: "Error setting <@U5> as pull-request reviewer: no github account for slack user <@U5>",
		},
		{
			name:         "calls assign and review for an individual",
			inputMessage: "<@U1234> assign <@G1> for https://github.com/Clever/fake-repo/pull/1 https://github.com/Clever/fake-repo2/pull/1",
			expectedUser: testGithubUser.SlackID,
			expectations: func(mocks *BotMocks) {
				gomock.InOrder(
					mocks.WhoIsWhoClient.EXPECT().UserBySlackID("G1").Return(testGithubUser, nil),
					// check github calls
					mocks.GithubClient.EXPECT().AddAssignees(gomock.Any(), testGithubOrg, "fake-repo", 1, gomock.Any()),
					mocks.GithubClient.EXPECT().AddReviewers(gomock.Any(), testGithubOrg, "fake-repo", 1, gomock.Any()),
					mocks.GithubClient.EXPECT().AddAssignees(gomock.Any(), testGithubOrg, "fake-repo2", 1, gomock.Any()),
					mocks.GithubClient.EXPECT().AddReviewers(gomock.Any(), testGithubOrg, "fake-repo2", 1, gomock.Any()),
				)
			},
			expectedMessage: "Set <@G1> as pull-request reviewer",
		},
	} {
		t.Logf("Case: %s. Input: %s", test.name, test.inputMessage)
		mockbot, mocks, mockCtrl := getMockBot(t)
		defer mockCtrl.Finish()

		mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
		test.expectations(mocks)
		message := makeSlackOutgoingMessage(test.expectedMessage)
		mocks.SlackRTM.EXPECT().NewOutgoingMessage(test.expectedMessage, testChannel).Return(message)
		mocks.SlackRTM.EXPECT().SendMessage(message)

		mockbot.DecodeMessage(makeSlackMessage(test.inputMessage))
	}
}

func TestPickIndividualInvalidName(t *testing.T) {
	mockbot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
	msg := didNotUnderstand
	message := makeSlackOutgoingMessage(msg)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
	mocks.SlackRTM.EXPECT().SendMessage(message)

	mockbot.DecodeMessage(makeSlackMessage("<@U1234> pick @not-a-user"))
}

func TestPickUserNoUserError(t *testing.T) {
	mockbot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
	msg := pickUserProblem
	message := makeSlackOutgoingMessage(msg)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
	mocks.SlackRTM.EXPECT().SendMessage(message)

	mockbot.DecodeMessage(makeSlackMessage("<@U1234> pick a eng-empty-team"))
}

func TestPickUserNoUserErrorDueToOmit(t *testing.T) {
	mockbot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
	msg := pickUserProblem
	message := makeSlackOutgoingMessage(msg)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
	mocks.SlackRTM.EXPECT().SendMessage(message)

	mockbot.DecodeMessage(makeSlackMessage("<@U1234> pick a eng-same-user-team"))
}

func TestAddOverride(t *testing.T) {
	mockbot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	t.Log("Can set override to add a user to a team")
	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
	msg := "Added <@U5555> to team example-team! Remember to update https://github.com/orgs/Clever/eng-example-team/edit/review_assignment too!"
	message := makeSlackOutgoingMessage(msg)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
	mocks.SlackRTM.EXPECT().SendMessage(message)
	mocks.WhoIsWhoClient.EXPECT().UserBySlackID("U5555")
	mocks.WhoIsWhoClient.EXPECT().UpsertUser("pickabot", gomock.Any())

	assert.Equal(t, 0, len(mockbot.TeamOverrides))
	mockbot.DecodeMessage(makeSlackMessage("<@U1234> <@U5555> is an eng-example-team"))
	assert.Equal(t, []Override{
		Override{
			User:    whoswho.User{SlackID: "U5555"},
			Team:    "example-team",
			Include: true,
		},
	}, mockbot.TeamOverrides)

	t.Log("Can set override to remove a user from a team")
	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
	msg2 := "Removed <@U7777> from team example-team! Remember to update https://github.com/orgs/Clever/eng-example-team/edit/review_assignment too!"
	message2 := makeSlackOutgoingMessage(msg2)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg2, testChannel).Return(message2)
	mocks.SlackRTM.EXPECT().SendMessage(message2)
	mocks.WhoIsWhoClient.EXPECT().UserBySlackID("U7777")
	mocks.WhoIsWhoClient.EXPECT().UpsertUser("pickabot", gomock.Any())

	mockbot.DecodeMessage(makeSlackMessage("<@U1234> <@U7777> is not eng-example-team"))
	assert.Equal(t, []Override{
		Override{
			User:    whoswho.User{SlackID: "U5555"},
			Team:    "example-team",
			Include: true,
		},
		Override{
			User:    whoswho.User{SlackID: "U7777"},
			Team:    "example-team",
			Include: false,
		},
	}, mockbot.TeamOverrides)

	t.Log("Can update a user's override")
	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
	msg3 := "Added <@U7777> to team example-team! Remember to update https://github.com/orgs/Clever/eng-example-team/edit/review_assignment too!"
	message3 := makeSlackOutgoingMessage(msg3)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg3, testChannel).Return(message3)
	mocks.SlackRTM.EXPECT().SendMessage(message3)
	mocks.WhoIsWhoClient.EXPECT().UserBySlackID("U7777")
	mocks.WhoIsWhoClient.EXPECT().UpsertUser("pickabot", gomock.Any())

	mockbot.DecodeMessage(makeSlackMessage("<@U1234> <@U7777> is an eng-example-team"))
	assert.Equal(t, []Override{
		Override{
			User:    whoswho.User{SlackID: "U5555"},
			Team:    "example-team",
			Include: true,
		},
		Override{
			User:    whoswho.User{SlackID: "U7777"},
			Team:    "example-team",
			Include: true,
		},
	}, mockbot.TeamOverrides)

}

func TestAddOverrideAlternateMessageMatcher(t *testing.T) {
	mockbot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	t.Log("Can set override to add a user to a team")
	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
	msg := "Added <@U5555> to team example-team! Remember to update https://github.com/orgs/Clever/eng-example-team/edit/review_assignment too!"
	message := makeSlackOutgoingMessage(msg)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
	mocks.SlackRTM.EXPECT().SendMessage(message)
	mocks.WhoIsWhoClient.EXPECT().UserBySlackID("U5555")
	mocks.WhoIsWhoClient.EXPECT().UpsertUser("pickabot", gomock.Any())

	assert.Equal(t, 0, len(mockbot.TeamOverrides))
	mockbot.DecodeMessage(makeSlackMessage("<@U1234> add <@U5555> to eng-example-team"))
	assert.Equal(t, []Override{
		Override{
			User:    whoswho.User{SlackID: "U5555"},
			Team:    "example-team",
			Include: true,
		},
	}, mockbot.TeamOverrides)

	t.Log("Can set override to remove a user from a team")
	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
	msg2 := "Removed <@U7777> from team example-team! Remember to update https://github.com/orgs/Clever/eng-example-team/edit/review_assignment too!"
	message2 := makeSlackOutgoingMessage(msg2)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg2, testChannel).Return(message2)
	mocks.SlackRTM.EXPECT().SendMessage(message2)
	mocks.WhoIsWhoClient.EXPECT().UserBySlackID("U7777")
	mocks.WhoIsWhoClient.EXPECT().UpsertUser("pickabot", gomock.Any())

	mockbot.DecodeMessage(makeSlackMessage("<@U1234> remove <@U7777> from eng-example-team"))
	assert.Equal(t, []Override{
		Override{
			User:    whoswho.User{SlackID: "U5555"},
			Team:    "example-team",
			Include: true,
		},
		Override{
			User:    whoswho.User{SlackID: "U7777"},
			Team:    "example-team",
			Include: false,
		},
	}, mockbot.TeamOverrides)

	t.Log("Can update a user's override")
	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
	msg3 := "Added <@U7777> to team example-team! Remember to update https://github.com/orgs/Clever/eng-example-team/edit/review_assignment too!"
	message3 := makeSlackOutgoingMessage(msg3)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg3, testChannel).Return(message3)
	mocks.SlackRTM.EXPECT().SendMessage(message3)
	mocks.WhoIsWhoClient.EXPECT().UserBySlackID("U7777")
	mocks.WhoIsWhoClient.EXPECT().UpsertUser("pickabot", gomock.Any())

	mockbot.DecodeMessage(makeSlackMessage("<@U1234> add <@U7777> to eng-example-team"))
	assert.Equal(t, []Override{
		Override{
			User:    whoswho.User{SlackID: "U5555"},
			Team:    "example-team",
			Include: true,
		},
		Override{
			User:    whoswho.User{SlackID: "U7777"},
			Team:    "example-team",
			Include: true,
		},
	}, mockbot.TeamOverrides)

}

func TestAddFlair(t *testing.T) {
	mockbot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil)
	msg := "<@U0>, I like your style!"
	message := makeSlackOutgoingMessage(msg)
	mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message)
	mocks.SlackRTM.EXPECT().SendMessage(message)
	mocks.WhoIsWhoClient.EXPECT().UserBySlackID("U0")
	mocks.WhoIsWhoClient.EXPECT().UpsertUser("pickabot", gomock.Any())

	assert.Equal(t, 0, len(mockbot.UserFlair))
	assert.Equal(t, "", mockbot.UserFlair["U0"])
	mockbot.DecodeMessage(makeSlackMessage("<@U1234> add flair :dance:"))
	assert.Equal(t, ":dance:", mockbot.UserFlair["U0"])
}

func TestSetAssigneeWithEmptyGithubFromOverride(t *testing.T) {
	mockbot, mocks, mockCtrl := getMockBot(t)
	defer mockCtrl.Finish()

	// first we add a member to the team overrides
	userMsg := "<@U1234> add <@U7777> to eng-empty-team"
	msg := "Added <@U7777> to team empty-team! Remember to update https://github.com/orgs/Clever/eng-empty-team/edit/review_assignment too!"
	message := makeSlackOutgoingMessage(msg)
	// next we try to assign a pr
	userMsg2 := "<@U1234> assign a empty-team for https://github.com/Clever/fake-repo/pull/1"
	msg2 := "Set <@U7777> as pull-request reviewer"
	message2 := makeSlackOutgoingMessage(msg2)

	gomock.InOrder(
		// mocks for adding user
		mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil),
		mocks.WhoIsWhoClient.EXPECT().UserBySlackID("U7777"),
		mocks.WhoIsWhoClient.EXPECT().UpsertUser("pickabot", gomock.Any()),
		mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg, testChannel).Return(message),
		mocks.SlackRTM.EXPECT().SendMessage(message),
		// mocks for pick from empty-team
		mocks.SlackAPI.EXPECT().GetUserInfo("U1234").Return(makeSlackUser(testUserID), nil),
		// verify we try to look up the user by slack id from whoswho
		mocks.WhoIsWhoClient.EXPECT().UserBySlackID("U7777").Return(whoswho.User{Github: "7777", SlackID: "U7777"}, nil),
		// verify subsequent calls to github
		mocks.GithubClient.EXPECT().AddAssignees(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()),
		mocks.GithubClient.EXPECT().AddReviewers(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()),
		mocks.SlackRTM.EXPECT().NewOutgoingMessage(msg2, testChannel).Return(message2),
		mocks.SlackRTM.EXPECT().SendMessage(message2),
	)

	mockbot.DecodeMessage(makeSlackMessage(userMsg))
	assert.Equal(t, []Override{
		Override{
			User:    whoswho.User{SlackID: "U7777"},
			Team:    "empty-team",
			Include: true,
		},
	}, mockbot.TeamOverrides)

	mockbot.DecodeMessage(makeSlackMessage(userMsg2))
}
