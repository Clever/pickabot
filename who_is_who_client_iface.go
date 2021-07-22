package main

import whoswho "github.com/Clever/who-is-who/go-client"

// generate mocks of dependencies for use during testing
//go:generate bin/mockgen -package main -source $PWD/who_is_who_client_iface.go -destination who_is_who_client_mock_test.go whoIsWhoClientIface

type whoIsWhoClientIface interface {
	UserBySlackID(string) (whoswho.User, error)
	UpsertUser(string, whoswho.User) (whoswho.User, error)
	GetUserList() ([]whoswho.User, error)
}
