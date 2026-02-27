package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/browser"
)

const (
	ClientID = "8918"
	BaseURL  = "https://api.put.io/v2"
)

type oobCodeResponse struct {
	Code string `json:"code"`
}

type oobTokenResponse struct {
	OAuthToken string `json:"oauth_token"`
}

func Authenticate(ctx context.Context) (string, error) {
	code, err := getOOBCode()
	if err != nil {
		return "", fmt.Errorf("getting OOB code: %w", err)
	}

	approveURL := fmt.Sprintf("https://app.put.io/authenticate?client_id=%s&response_type=oob&oob_code=%s", ClientID, code)
	fmt.Printf("Opening browser for authentication...\n")
	fmt.Printf("If the browser doesn't open, visit:\n  %s\n\n", approveURL)
	_ = browser.OpenURL(approveURL)

	fmt.Println("Waiting for approval...")
	return pollForToken(ctx, code)
}

func getOOBCode() (string, error) {
	resp, err := http.Get(BaseURL + "/oauth2/oob/code?app_id=" + ClientID)
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
