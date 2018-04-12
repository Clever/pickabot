package github

import (
	"context"
	"fmt"
	"time"

	"github.com/Clever/kayvee-go/logger"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var (
	// Github's rate limit for authenticated requests is 5000 QPH = 83.3 QPM = 1.38 QPS = 720ms/query
	// We also use a global limiter to prevent concurrent requests, which trigger Github's abuse detection
	githubLimiter = time.NewTicker(720 * time.Millisecond)
)

// AppClientIface represents the endpoints available to a github application
type AppClientIface interface {
	AddAssignees(ctx context.Context, owner, repo string, number int, assignees []string) (*github.Issue, *github.Response, error)
}

// AppClient is an implementation of the AppClientIface
// auth reference: https://developer.github.com/apps/building-github-apps/authentication-options-for-github-apps
type AppClient struct {
	AppID          string
	InstallationID string
	Logger         logger.KayveeLogger
	PrivateKey     []byte

	jwt               Token
	githubAccessToken Token
	client            *github.Client
}

// AddAssignees adds assignees to an issue
func (a *AppClient) AddAssignees(ctx context.Context, owner, repo string, number int, assignees []string) (*github.Issue, *github.Response, error) {
	if err := a.checkClient(); err != nil {
		return &github.Issue{}, &github.Response{}, err
	}
	return a.client.Issues.AddAssignees(context.Background(), owner, repo, number, assignees)
}

// checkClient validates the current token and re-authenticates if it needs to
// this should be called BEFORE every call of the github client
func (a *AppClient) checkClient() error {
	if a.githubAccessToken.IsExpired() {
		if err := a.setupNewClient(); err != nil {
			return err
		}
	}

	<-githubLimiter.C
	return nil
}

// setupNewClient sets up authorization for a new github client
func (a *AppClient) setupNewClient() error {
	err := a.generateGithubAccessToken()
	if err != nil {
		return fmt.Errorf("error getting GitHub access token: %s", err)
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: a.githubAccessToken.Token})
	tc := oauth2.NewClient(context.Background(), ts)
	a.client = github.NewClient(tc)

	return nil
}
