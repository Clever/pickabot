package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMessageForPRs(t *testing.T) {
	assert := assert.New(t)

	// It only pulls github.com/Clever
	message := "pick a infra for http://google.com http://yahoo.com http://github.com/notclever"
	prs := parseMessageForPRs(message)
	assert.Equal(0, len(prs))

	// It filters out non-prs
	message = `pick a infra for 
		https://github.com/Clever/ark-config/pull/1 https://github.com/Clever/terrafam/pull/2 
		https://github.com/Clever/sd2`
	prs = parseMessageForPRs(message)
	assert.Equal(2, len(prs))
	assert.Contains(prs, githubPR{Owner: "Clever", Repo: "ark-config", PRNumber: 1})
	assert.Contains(prs, githubPR{Owner: "Clever", Repo: "terrafam", PRNumber: 2})
}
