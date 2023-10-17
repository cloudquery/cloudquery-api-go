package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/adrg/xdg"
)

const (
	FirebaseAPIKey         = "AIzaSyCxsrwjABEF-dWLzUqmwiL-ct02cnG9GCs"
	TokenBaseURL           = "https://securetoken.googleapis.com"
	EnvVarCloudQueryAPIKey = "CLOUDQUERY_API_KEY"
	ExpiryBuffer           = 60 * time.Second
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    string `json:"expires_in"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	UserID       string `json:"user_id"`
	ProjectID    string `json:"project_id"`
}

type TokenClient struct {
	url       string
	apiKey    string
	idToken   string
	expiresAt time.Time
}

func NewTokenClient() *TokenClient {
	return &TokenClient{
		url:    TokenBaseURL,
		apiKey: FirebaseAPIKey,
	}
}

// GetToken returns the ID token
// If CLOUDQUERY_API_KEY is set, it returns that value, otherwise it returns an ID token generated from the refresh token.
func (tc *TokenClient) GetToken() (string, error) {
	if token := os.Getenv(EnvVarCloudQueryAPIKey); token != "" {
		return token, nil
	}

	// If the token is not expired, return it
	if !tc.expiresAt.IsZero() && tc.expiresAt.Sub(time.Now().UTC()) > ExpiryBuffer {
		return tc.idToken, nil
	}

	refreshToken, err := ReadRefreshToken()
	if err != nil {
		return "", fmt.Errorf("failed to read refresh token: %w. Hint: You may need to run `cloudquery login` or set %s", err, EnvVarCloudQueryAPIKey)
	}
	if refreshToken == "" {
		return "", fmt.Errorf("authentication token not found. Hint: You may need to run `cloudquery login` or set %s", EnvVarCloudQueryAPIKey)
	}
	tokenResponse, err := tc.generateToken(refreshToken)
	if err != nil {
		return "", fmt.Errorf("failed to sign in with custom token: %w", err)
	}

	if err := SaveRefreshToken(tokenResponse.RefreshToken); err != nil {
		return "", fmt.Errorf("failed to save refresh token: %w", err)
	}

	if err := tc.updateIDToken(tokenResponse); err != nil {
		return "", fmt.Errorf("failed to update ID token: %w", err)
	}

	return tc.idToken, nil
}

func (tc *TokenClient) generateToken(refreshToken string) (*tokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	resp, err := http.PostForm(fmt.Sprintf("%s/v1/token?key=%s", tc.url, tc.apiKey), data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read response body: %w", readErr)
		}
		return nil, fmt.Errorf("failed to refresh token: %s: %s", resp.Status, body)
	}

	var tr tokenResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := parseToken(body, &tr); err != nil {
		return nil, err
	}

	return &tr, nil
}

func (tc *TokenClient) updateIDToken(tr *tokenResponse) error {
	// Convert string duration in seconds to time.Duration
	duration, err := time.ParseDuration(tr.ExpiresIn + "s")
	if err != nil {
		return err
	}

	tc.expiresAt = time.Now().UTC().Add(duration)
	tc.idToken = tr.IDToken
	return nil
}

func parseToken(response []byte, tr *tokenResponse) error {
	err := json.Unmarshal(response, tr)
	if err != nil {
		return err
	}
	return nil
}

// SaveRefreshToken saves the refresh token to the token file
func SaveRefreshToken(refreshToken string) error {
	tokenFilePath, err := xdg.DataFile("cloudquery/token")
	if err != nil {
		return fmt.Errorf("failed to get token file path: %w", err)
	}
	tokenFile, err := os.OpenFile(tokenFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open token file %q for writing: %w", tokenFilePath, err)
	}
	defer func() {
		if closeErr := tokenFile.Close(); closeErr != nil {
			fmt.Printf("error closing token file: %v", closeErr)
		}
	}()
	if _, err = tokenFile.WriteString(refreshToken); err != nil {
		return fmt.Errorf("failed to write token to %q: %w", tokenFilePath, err)
	}
	return nil
}

// ReadRefreshToken reads the refresh token from the token file
func ReadRefreshToken() (string, error) {
	tokenFilePath, err := xdg.DataFile("cloudquery/token")
	if err != nil {
		return "", fmt.Errorf("failed to get token file path: %w", err)
	}
	b, err := os.ReadFile(tokenFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read token file: %w", err)
	}
	return strings.TrimSpace(string(b)), nil
}

// RemoveRefreshToken removes the token file
func RemoveRefreshToken() error {
	tokenFilePath, err := xdg.DataFile("cloudquery/token")
	if err != nil {
		return fmt.Errorf("failed to get token file path: %w", err)
	}
	if err := os.RemoveAll(tokenFilePath); err != nil {
		return fmt.Errorf("failed to remove token file %q: %w", tokenFilePath, err)
	}
	return nil
}
