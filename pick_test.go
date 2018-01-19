package main

import (
	"math/rand"
	"testing"

	whoswho "github.com/Clever/who-is-who/go-client"
	"github.com/stretchr/testify/assert"
)

// set random source in unit tests, so they run predictably
var source = rand.NewSource(0)

func TestPickUser(t *testing.T) {
	assert := assert.New(t)

	currentUser := whoswho.User{
		SlackID: "U99995",
	}
	u1 := whoswho.User{SlackID: "U99991"}
	u2 := whoswho.User{SlackID: "U99992"}
	users := []whoswho.User{currentUser, u1, u2}
	omit := &currentUser
	picked, err := pickUser(users, omit, source)
	assert.NoError(err)
	assert.Equal(u1, picked)
}

func TestPickUserFailsIfEmptyUserList(t *testing.T) {
	assert := assert.New(t)

	t.Log("Fails if empty list")
	users := []whoswho.User{}
	_, err := pickUser(users, nil, source)
	assert.Error(err)
	assert.Equal(ErrNoUsers, err)

	t.Log("Fails if empty list due to omitted user")
	currentUser := whoswho.User{
		SlackID: "U99995",
	}
	users = []whoswho.User{currentUser}
	omit := &currentUser
	_, err = pickUser(users, omit, source)
	assert.Error(err)
	assert.Equal(ErrNoUsers, err)
}

func TestPickUserDedupsBySlackID(t *testing.T) {
	assert := assert.New(t)

	currentUser := whoswho.User{
		SlackID: "U99995",
	}
	u1 := whoswho.User{SlackID: "U99991"}
	u2 := whoswho.User{SlackID: "U99992"}
	users := []whoswho.User{currentUser, currentUser, u1, u2, u2, u2, u2, u2, u2, u2, u2, u2, u2, u2}
	omit := &currentUser
	picked, err := pickUser(users, omit, source)
	assert.NoError(err)
	assert.Equal(u1, picked)
}
