package github

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/knq/jwt"
	"github.com/knq/pemutil"
)

// Token represents an authorization claim
type Token struct {
	Token      string
	Expiration time.Time
}

// IsExpired returns whether the token is still valid
func (t Token) IsExpired() bool {
	return time.Now().After(t.Expiration)
}

func generateNewJWT(appID, pemFileName string) (*Token, error) {
	// load key
	keyset, err := pemutil.LoadFile(pemFileName)
	if err != nil {
		return nil, err
	}

	// create RSA256 using keyset
	rsa256, err := jwt.RS256.New(keyset)
	if err != nil {
		return nil, err
	}

	// setup our claims
	currentTime := time.Now()
	expr := currentTime.Add(10 * time.Minute)
	claims := jwt.Claims{
		Issuer:     appID,
		IssuedAt:   json.Number(strconv.FormatInt(currentTime.Unix(), 10)),
		Expiration: json.Number(strconv.FormatInt(expr.Unix(), 10)),
	}
	fmt.Printf("Generating jwt with claims: %+v\n", claims)

	buf, err := rsa256.Encode(&claims)
	if err != nil {
		return nil, err
	}

	return &Token{
		Token:      string(buf[:]),
		Expiration: expr,
	}, nil
}

func generateGithubAccessToken(token, installationID string) (*Token, error) {
	client := &http.Client{}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://api.github.com/installations/%s/access_tokens", installationID), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	// We need to pass this "machine-man" media type for our expected format
	// https://developer.github.com/v3/media/
	// https://developer.github.com/apps/building-github-apps/authentication-options-for-github-apps/#authenticating-as-an-installation
	req.Header.Set("Accept", "application/vnd.github.machine-man-preview+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("received failing status code: %d", resp.StatusCode)
	}

	var outputToken Token
	err = json.Unmarshal(body, &outputToken)
	if err != nil {
		return nil, err
	}

	return &outputToken, nil
}
