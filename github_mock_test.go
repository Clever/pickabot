// Code generated by MockGen. DO NOT EDIT.
// Source: /Users/sayansamanta/go/src/github.com/Clever/pickabot/github/client.go

// Package main is a generated GoMock package.
package main

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	github "github.com/google/go-github/github"
)

// MockAppClientIface is a mock of AppClientIface interface
type MockAppClientIface struct {
	ctrl     *gomock.Controller
	recorder *MockAppClientIfaceMockRecorder
}

// MockAppClientIfaceMockRecorder is the mock recorder for MockAppClientIface
type MockAppClientIfaceMockRecorder struct {
	mock *MockAppClientIface
}

// NewMockAppClientIface creates a new mock instance
func NewMockAppClientIface(ctrl *gomock.Controller) *MockAppClientIface {
	mock := &MockAppClientIface{ctrl: ctrl}
	mock.recorder = &MockAppClientIfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockAppClientIface) EXPECT() *MockAppClientIfaceMockRecorder {
	return m.recorder
}

// AddAssignees mocks base method
func (m *MockAppClientIface) AddAssignees(ctx context.Context, owner, repo string, number int, assignees []string) (*github.Issue, *github.Response, error) {
	ret := m.ctrl.Call(m, "AddAssignees", ctx, owner, repo, number, assignees)
	ret0, _ := ret[0].(*github.Issue)
	ret1, _ := ret[1].(*github.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// AddAssignees indicates an expected call of AddAssignees
func (mr *MockAppClientIfaceMockRecorder) AddAssignees(ctx, owner, repo, number, assignees interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddAssignees", reflect.TypeOf((*MockAppClientIface)(nil).AddAssignees), ctx, owner, repo, number, assignees)
}

// AddReviewers mocks base method
func (m *MockAppClientIface) AddReviewers(ctx context.Context, owner, repo string, number int, reviewers []string) (*github.PullRequest, *github.Response, error) {
	ret := m.ctrl.Call(m, "AddReviewers", ctx, owner, repo, number, reviewers)
	ret0, _ := ret[0].(*github.PullRequest)
	ret1, _ := ret[1].(*github.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// AddReviewers indicates an expected call of AddReviewers
func (mr *MockAppClientIfaceMockRecorder) AddReviewers(ctx, owner, repo, number, reviewers interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddReviewers", reflect.TypeOf((*MockAppClientIface)(nil).AddReviewers), ctx, owner, repo, number, reviewers)
}
