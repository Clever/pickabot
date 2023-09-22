// Code generated by MockGen. DO NOT EDIT.
// Source: slackapi/SlackService.go

// Package main is a generated GoMock package.
package main

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	slack "github.com/slack-go/slack"
)

// MockSlackAPIService is a mock of SlackAPIService interface.
type MockSlackAPIService struct {
	ctrl     *gomock.Controller
	recorder *MockSlackAPIServiceMockRecorder
}

// MockSlackAPIServiceMockRecorder is the mock recorder for MockSlackAPIService.
type MockSlackAPIServiceMockRecorder struct {
	mock *MockSlackAPIService
}

// NewMockSlackAPIService creates a new mock instance.
func NewMockSlackAPIService(ctrl *gomock.Controller) *MockSlackAPIService {
	mock := &MockSlackAPIService{ctrl: ctrl}
	mock.recorder = &MockSlackAPIServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSlackAPIService) EXPECT() *MockSlackAPIServiceMockRecorder {
	return m.recorder
}

// GetUserInfo mocks base method.
func (m *MockSlackAPIService) GetUserInfo(user string) (*slack.User, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUserInfo", user)
	ret0, _ := ret[0].(*slack.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUserInfo indicates an expected call of GetUserInfo.
func (mr *MockSlackAPIServiceMockRecorder) GetUserInfo(user interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUserInfo", reflect.TypeOf((*MockSlackAPIService)(nil).GetUserInfo), user)
}

// NewRTM mocks base method.
func (m *MockSlackAPIService) NewRTM() *slack.RTM {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewRTM")
	ret0, _ := ret[0].(*slack.RTM)
	return ret0
}

// NewRTM indicates an expected call of NewRTM.
func (mr *MockSlackAPIServiceMockRecorder) NewRTM() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewRTM", reflect.TypeOf((*MockSlackAPIService)(nil).NewRTM))
}

// MockSlackRTMService is a mock of SlackRTMService interface.
type MockSlackRTMService struct {
	ctrl     *gomock.Controller
	recorder *MockSlackRTMServiceMockRecorder
}

// MockSlackRTMServiceMockRecorder is the mock recorder for MockSlackRTMService.
type MockSlackRTMServiceMockRecorder struct {
	mock *MockSlackRTMService
}

// NewMockSlackRTMService creates a new mock instance.
func NewMockSlackRTMService(ctrl *gomock.Controller) *MockSlackRTMService {
	mock := &MockSlackRTMService{ctrl: ctrl}
	mock.recorder = &MockSlackRTMServiceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSlackRTMService) EXPECT() *MockSlackRTMServiceMockRecorder {
	return m.recorder
}

// NewOutgoingMessage mocks base method.
func (m *MockSlackRTMService) NewOutgoingMessage(text, channel string) *slack.OutgoingMessage {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewOutgoingMessage", text, channel)
	ret0, _ := ret[0].(*slack.OutgoingMessage)
	return ret0
}

// NewOutgoingMessage indicates an expected call of NewOutgoingMessage.
func (mr *MockSlackRTMServiceMockRecorder) NewOutgoingMessage(text, channel interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewOutgoingMessage", reflect.TypeOf((*MockSlackRTMService)(nil).NewOutgoingMessage), text, channel)
}

// SendMessage mocks base method.
func (m *MockSlackRTMService) SendMessage(msg *slack.OutgoingMessage) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SendMessage", msg)
}

// SendMessage indicates an expected call of SendMessage.
func (mr *MockSlackRTMServiceMockRecorder) SendMessage(msg interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMessage", reflect.TypeOf((*MockSlackRTMService)(nil).SendMessage), msg)
}
