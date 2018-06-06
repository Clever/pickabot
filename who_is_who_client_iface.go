package main

import whoswho "github.com/Clever/who-is-who/go-client"

type whoIsWhoClientIface interface {
	UserBySlackID(string) (whoswho.User, error)
	UpsertUser(string, whoswho.User) (whoswho.User, error)
	GetUserList() ([]whoswho.User, error)
}
