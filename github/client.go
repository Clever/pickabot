package github

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/github"
)

var (
	// Github's rate limit for authenticated requests is 5000 QPH = 83.3 QPM = 1.38 QPS = 720ms/query
	// We also use a global limiter to prevent concurrent requests, which trigger Github's abuse detection
	githubLimiter = time.NewTicker(720 * time.Millisecond)
)

// AppClient represents the endpoints available to a github application
type AppClient interface {
	AddAssignees(ctx context.Context, owner, repo string, number int, assignees []string) (*github.Issue, *github.Response, error)
}

// AppClientImpl is an implementation of the AppClient
type AppClientImpl struct {
	AppID          string
	InstallationID string
	PrivateKey     []byte

	jwt               *Token
	githubAccessToken *Token
	client            *github.Client
}

func (a *AppClientSt) checkClient() error {
	if a.currentJWT.IsExpired() {
		a.currentJWT, err = generateNewJWT(appID, "pickabot.private-key.pem")
		if err != nil {
			return fmt.Errorf("error generating JWT for GitHub access: %s", err)
		}
	}
	if a.currentGithubToken.IsExpired() {
		currentGithubToken, err = generateGithubAccessToken(currentJWT.Token, installationID)
		if err != nil {
			return fmt.Errorf("error getting GitHub access token: %s", err)
		}
	}
}
