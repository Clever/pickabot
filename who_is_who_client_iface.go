package main

import whoswho "github.com/Clever/who-is-who/go-client"

// generate mocks of dependencies for use during testing
//go:generate sh -c "$PWD/bin/mockgen -package main -source $PWD/who_is_who_client_iface.go whoIsWhoClientIface > who_is_who_client_mock_test.go"

type whoIsWhoClientIface interface {
	UserBySlackID(string) (whoswho.User, error)
	UpsertUser(string, whoswho.User) (whoswho.User, error)
	GetUserList() ([]whoswho.User, error)
}
