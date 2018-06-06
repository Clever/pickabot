package main

import (
	"fmt"
	"strconv"
	"strings"

	"mvdan.cc/xurls"
)

type githubPR struct {
	Owner    string
	Repo     string
	PRNumber int
}

// parseMessageForPRs searchs for strings matching: github.com/{ORG_NAME}/{REPO}/pull/...
func parseMessageForPRs(githubOrg, message string) []githubPR {
	var prs []githubPR
	githubURLMatcher, _ := xurls.StrictMatchingScheme(fmt.Sprintf("github.com/%s", githubOrg))
	urls := githubURLMatcher.FindAllString(message, -1)

	for _, url := range urls {
		urlParts := strings.Split(url, "/")
		// length checks and action check
		if len(urlParts) < 5 || urlParts[3] != "pull" {
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
