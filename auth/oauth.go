package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/browser"
)

const BaseURL = "https://api.put.io/v2"

// ErrNoClientID means no put.io OAuth app ID is configured.
var ErrNoClientID = errors.New(`no put.io app ID set: create an OAuth app in put.io (Settings → OAuth Apps), ` +
	`then set "client_id" in ~/.config/teleput/config.json or PUTIO_CLIENT_ID`)

type oobCodeResponse struct {
	Code string `json:"code"`
}

type oobTokenResponse struct {
	OAuthToken string `json:"oauth_token"`
}

// Authenticate runs put.io's out-of-band sign-in for the OAuth app clientID
// and returns the approved token.
func Authenticate(ctx context.Context, clientID string) (string, error) {
	if clientID == "" {
		return "", ErrNoClientID
	}
	code, err := getOOBCode(clientID)
	if err != nil {
		return "", fmt.Errorf("getting OOB code: %w", err)
	}

	// The code must be linked via put.io/link; the /authenticate page only
	// displays a token and never attaches it to the OOB code we poll.
	approveURL := fmt.Sprintf("https://app.put.io/link?code=%s", code)
	fmt.Printf("Opening browser for authentication...\n")
	fmt.Printf("If the browser doesn't open, visit:\n  %s\n(or enter code %s at put.io/link)\n\n", approveURL, code)
	_ = browser.OpenURL(approveURL)

	fmt.Println("Waiting for approval...")
	return pollForToken(ctx, code)
}

func getOOBCode(clientID string) (string, error) {
	resp, err := http.Get(BaseURL + "/oauth2/oob/code?app_id=" + url.QueryEscape(clientID))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}

	var result oobCodeResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	return result.Code, nil
}

func pollForToken(ctx context.Context, code string) (string, error) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			return "", fmt.Errorf("authentication timed out after 5 minutes")
		case <-ticker.C:
			token, err := checkCode(code)
			if err != nil {
				continue
			}
			if token != "" {
				return token, nil
			}
		}
	}
}

func checkCode(code string) (string, error) {
	resp, err := http.Get(BaseURL + "/oauth2/oob/code/" + code)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("not yet approved")
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result oobTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if result.OAuthToken == "" {
		return "", fmt.Errorf("no token yet")
	}
	return result.OAuthToken, nil
}
