package github

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/Clever/catapult/logger"
	"github.com/knq/jwt"
	"github.com/knq/pemutil"
)

// Token represents an authorization claim
type Token struct {
	Token      string
	Expiration time.Time
}

// IsExpired returns whether the token is still valid
// we give ourselves a 30 second buffer to give the subsequent requests to the Github API
// time to process
func (t Token) IsExpired() bool {
	return time.Now().Add(-1 * 30 * time.Second).After(t.Expiration)
}

func (a *AppClientImpl) generateNewJWT() error {
	// load key
	keyset, err := pemutil.DecodeBytes(a.PrivateKey)
	if err != nil {
		return err
	}

	// create RSA256 using keyset
	rsa256, err := jwt.RS256.New(keyset)
	if err != nil {
		return err
	}

	// setup our claims
	currentTime := time.Now()
	expr := currentTime.Add(10 * time.Minute)
	claims := jwt.Claims{
		Issuer:     a.AppID,
		IssuedAt:   json.Number(strconv.FormatInt(currentTime.Unix(), 10)),
		Expiration: json.Number(strconv.FormatInt(expr.Unix(), 10)),
	}

	buf, err := rsa256.Encode(&claims)
	if err != nil {
		return err
	}

	a.jwt = Token{
		Token:      string(buf[:]),
		Expiration: expr,
	}
	a.Logger.InfoD("generated-jwt-token", logger.M{"expiration": a.jwt.Expiration})
	return nil
}

func (a *AppClientImpl) generateGithubAccessToken() error {
	// check and re-generate JWT
	if a.jwt.IsExpired() {
		err := a.generateNewJWT()
		if err != nil {
			return fmt.Errorf("error generating JWT for GitHub access: %s", err)
		}
	}

	// ask for a token
	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.github.com/installations/%s/access_tokens", a.InstallationID), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", a.jwt.Token))
	// We need to pass this "machine-man" media type for our expected format
	// https://developer.github.com/v3/media/
	// https://developer.github.com/apps/building-github-apps/authentication-options-for-github-apps/#authenticating-as-an-installation
	req.Header.Set("Accept", "application/vnd.github.machine-man-preview+json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	// additionally check status code
	if resp.StatusCode >= 400 {
		return fmt.Errorf("received failing status code: %d", resp.StatusCode)
	}

	// parse response
	type githubToken struct {
		Token      string    `json:"token,omitempty"`
		Expiration time.Time `json:"expires_at,omitempty"`
	}
	var outputToken githubToken
	err = json.Unmarshal(body, &outputToken)
	if err != nil {
		return err
	}

	// yay we have a new token!
	a.githubAccessToken = Token{
		Token:      outputToken.Token,
		Expiration: outputToken.Expiration,
	}
	a.Logger.InfoD("generated-bearer-token", logger.M{"expiration": a.githubAccessToken.Expiration})
	return nil
}
