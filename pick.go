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
	seen := map[string]struct{}{}
	// handle dups
	dedupedUsers := []whoswho.User{}
	for _, u := range users {
		// Dedup based on SlackID
		_, ok := seen[u.SlackID]
		if !ok {
			dedupedUsers = append(dedupedUsers, u)
			seen[u.SlackID] = struct{}{}
		}
	}

	// handle omit
	if omit != nil {
		for idx, u := range dedupedUsers {
			// Remove omitted user
			if u.SlackID == omit.SlackID {
				dedupedUsers = append(dedupedUsers[:idx], dedupedUsers[idx+1:]...)
				break
			}
		}
	}

	if len(dedupedUsers) == 0 {
		return whoswho.User{}, ErrNoUsers
	}

	choice := rand.New(source).Intn(len(dedupedUsers))
	return dedupedUsers[choice], nil
}
