package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMessageForPRs(t *testing.T) {
	assert := assert.New(t)

	t.Log("Validates Github and org name")
	message := "pick a team for http://google.com http://yahoo.com http://github.com/msn"
	prs := parseMessageForPRs("pack", message)
	assert.Equal(0, len(prs))

	t.Log("Filters out non-prs")
	message = `pick a team member for 
		https://github.com/test/repo1/pull/1 https://github.com/test/repo2/pull/2/files
		https://github.com/test/not-a-pr`
	prs = parseMessageForPRs("test", message)
	assert.Equal(2, len(prs))
	assert.Contains(prs, githubPR{Owner: "test", Repo: "repo1", PRNumber: 1})
	assert.Contains(prs, githubPR{Owner: "test", Repo: "repo2", PRNumber: 2})
}
