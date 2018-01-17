package main

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

// set random source in unit tests, so they run predictably
var source = rand.NewSource(0)

func TestPickUser(t *testing.T) {
	assert := assert.New(t)

	currentUser := User{
		SlackHandle: "@n",
	}
	u1 := User{SlackHandle: "@user1"}
	u2 := User{SlackHandle: "@user2"}
	users := []User{currentUser, u1, u2}
	omit := &currentUser
	picked, err := pickUser(users, omit, source)
	assert.NoError(err)
	assert.Equal(u1, picked)
}

func TestPickUserFailsIfEmptyUserList(t *testing.T) {
	assert := assert.New(t)

	t.Log("Fails if empty list")
	users := []User{}
	_, err := pickUser(users, nil, source)
	assert.Error(err)
	assert.Equal(ErrNoUsers, err)

	t.Log("Fails if empty list due to omitted user")
	currentUser := User{
		SlackHandle: "@n",
	}
	users = []User{currentUser}
	omit := &currentUser
	_, err = pickUser(users, omit, source)
	assert.Error(err)
	assert.Equal(ErrNoUsers, err)
}
