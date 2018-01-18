package main

import (
	"errors"
	"math/rand"

	whoswho "github.com/Clever/who-is-who/go-client"
)

// ErrNoUsers occurs when there are no users to choose from
var ErrNoUsers = errors.New("no users to choose from")

// User model
type User struct {
	SlackHandle string
}

// pickUser chooses a User from the the list of users
// if omit is non-nil, it will omit that user from any response
func pickUser(users []whoswho.User, omit *whoswho.User, source rand.Source) (whoswho.User, error) {
	// handle omit
	if omit != nil {
		for idx, u := range users {
			// Remove omitted user
			if u.SlackID == omit.SlackID {
				users = append(users[:idx], users[idx+1:]...)
				break
			}
		}
	}

	if len(users) == 0 {
		return whoswho.User{}, ErrNoUsers
	}

	choice := rand.New(source).Intn(len(users))
	return users[choice], nil
}
