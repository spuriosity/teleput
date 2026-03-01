package auth

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/pkg/browser"
)

const (
	ClientID = "8918"
	BaseURL  = "https://api.put.io/v2"
)

type oobCodeResponse struct {
	Code string `json:"code"`
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

	fmt.Print("Paste your token here: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading token: %w", err)
	}
	token := strings.TrimSpace(line)
	if token == "" {
		return "", fmt.Errorf("no token provided")
	}
	return token, nil
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
