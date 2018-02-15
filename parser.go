package main

import (
	"strconv"
	"strings"

	"mvdan.cc/xurls"
)

// For now we will only match repos owned by Clever
var githubURLMatcher, _ = xurls.StrictMatchingScheme("github.com/Clever")

type githubPR struct {
	Owner    string
	Repo     string
	PRNumber int
}

func parseMessageForPRs(message string) []githubPR {
	var prs []githubPR
	urls := githubURLMatcher.FindAllString(message, -1)

	for _, url := range urls {
		urlParts := strings.Split(url, "/")
		// length checks and action check
		if len(urlParts) != 5 || urlParts[3] != "pull" {
			continue
		}
		prNumber, err := strconv.Atoi(urlParts[4])
		if err != nil {
			continue
		}
		prs = append(prs, githubPR{
			Owner:    urlParts[1],
			Repo:     urlParts[2],
			PRNumber: prNumber,
		})
	}

	return prs
}
