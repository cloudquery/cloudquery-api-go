package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/cloudquery/cloudquery-api-go/config"
)

const (
	FirebaseAPIKey         = "AIzaSyCxsrwjABEF-dWLzUqmwiL-ct02cnG9GCs"
	TokenBaseURL           = "https://securetoken.googleapis.com"
	EnvVarCloudQueryAPIKey = "CLOUDQUERY_API_KEY"
	ExpiryBuffer           = 60 * time.Second
	tokenFilePath          = "cloudquery/token"
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

type TokenType int

const (
	Undefined TokenType = iota
	BearerToken
	APIKey
)

var UndefinedToken = Token{Type: Undefined, Value: ""}

type Token struct {
	Type  TokenType
	Value string
}

func (t Token) String() string {
	return t.Value
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
func (tc *TokenClient) GetToken() (Token, error) {
	if tc.GetTokenType() == APIKey {
		return Token{Type: APIKey, Value: os.Getenv(EnvVarCloudQueryAPIKey)}, nil
	}

	// If the token is not expired, return it
	if !tc.expiresAt.IsZero() && tc.expiresAt.Sub(time.Now().UTC()) > ExpiryBuffer {
		return Token{Type: BearerToken, Value: tc.idToken}, nil
	}

	refreshToken, err := ReadRefreshToken()
	if err != nil {
		return UndefinedToken, fmt.Errorf("failed to read refresh token: %w. Hint: You may need to run `cloudquery login` or set %s", err, EnvVarCloudQueryAPIKey)
	}
	if refreshToken == "" {
		return UndefinedToken, fmt.Errorf("authentication token not found. Hint: You may need to run `cloudquery login` or set %s", EnvVarCloudQueryAPIKey)
	}
	tokenResponse, err := tc.generateToken(refreshToken)
	if err != nil {
		return UndefinedToken, fmt.Errorf("failed to sign in with custom token: %w", err)
	}

	if err := SaveRefreshToken(tokenResponse.RefreshToken); err != nil {
		return UndefinedToken, fmt.Errorf("failed to save refresh token: %w", err)
	}

	if err := tc.updateIDToken(tokenResponse); err != nil {
		return UndefinedToken, fmt.Errorf("failed to update ID token: %w", err)
	}

	return Token{Type: BearerToken, Value: tc.idToken}, nil
}

// GetTokenType returns the type of token that will be returned by GetToken
func (tc *TokenClient) GetTokenType() TokenType {
	if token := os.Getenv(EnvVarCloudQueryAPIKey); token != "" {
		return APIKey
	}
	return BearerToken
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
	return config.SaveDataString(tokenFilePath, refreshToken)
}

// ReadRefreshToken reads the refresh token from the token file
func ReadRefreshToken() (string, error) {
	return config.ReadDataString(tokenFilePath)
}

// RemoveRefreshToken removes the token file
func RemoveRefreshToken() error {
	return config.DeleteDataString(tokenFilePath)
}
