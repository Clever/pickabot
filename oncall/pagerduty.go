package oncall

// Things to do:
// 1. Client to interact with PagerDuty (https://godoc.org/github.com/PagerDuty/go-pagerduty#NewOAuthClient)
// 2. Call ListOnCallUsers (https://godoc.org/github.com/PagerDuty/go-pagerduty#Client.ListOnCallUsers)
// 3. Somehow merge the two lists (ListOnCallUsers vs who-is-who's representation)?
// 4. Return this to be stored within the bot's structure.

import (
	pagerduty "github.com/PagerDuty/go-pagerduty"
)

// generate mocks of dependencies for use during testing
//go:generate sh -c "$PWD/bin/mockgen -package main -source $PWD/pagerduty.go PagerDutyClientInterface > pagerduty_client_mock_test.go"

// PagerDutyClientInterface represents the endpoints available to a Pagerduty Client
type PagerDutyClientInterface interface {
	ListSchedules(pagerduty.ListSchedulesOptions) (*pagerduty.ListSchedulesResponse, error)
	ListOnCallUsers(string, pagerduty.ListOnCallUsersOptions) ([]pagerduty.User, error)
	ListOnCalls(pagerduty.ListOnCallOptions) (*pagerduty.ListOnCallsResponse, error)
}

// PagerDutyClient is an implementation of the PagerDutyClientInterface.
type PagerDutyClient struct {
	client *pagerduty.Client
}

// ListOnCalls shows all the oncalls.
func (p *PagerDutyClient) ListOnCalls(o pagerduty.ListOnCallOptions) (*pagerduty.ListOnCallsResponse, error) {
	return p.client.ListOnCalls(o)
}

// ListSchedules gets the schedules in PagerDuty.
func (p *PagerDutyClient) ListSchedules(o pagerduty.ListSchedulesOptions) (*pagerduty.ListSchedulesResponse, error) {
	return p.client.ListSchedules(o)
}
