package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshToken_RoundTrip(t *testing.T) {
	token := "my_token"

	err := SaveRefreshToken(token)
	require.NoError(t, err)

	readToken, err := ReadRefreshToken()
	require.NoError(t, err)

	require.Equal(t, token, readToken)
}

func TestRefreshToken_Removal(t *testing.T) {
	token := "my_token"

	err := SaveRefreshToken(token)
	require.NoError(t, err)

	_, err = ReadRefreshToken()
	require.NoError(t, err)

	err = RemoveRefreshToken()
	require.NoError(t, err)

	_, err = ReadRefreshToken()
	require.Error(t, err)
}

func TestToken_Stringer(t *testing.T) {
	token := Token{Type: BearerToken, Value: "my_token"}
	out := fmt.Sprintf("Bearer %s", token)
	require.Equal(t, "Bearer my_token", out)
}

func TestTokenClient_EnvironmentVariable(t *testing.T) {
	reset := overrideEnvironmentVariable(t, EnvVarCloudQueryAPIKey, "my_token")
	defer reset()

	token, err := NewTokenClient().GetToken()
	require.NoError(t, err)

	require.Equal(t, Token{Type: APIKey, Value: "my_token"}, token)
}

func TestTokenClient_GetToken_ShortExpiry(t *testing.T) {
	server, closer := fakeAuthServer(t, "0")
	defer closer()

	err := SaveRefreshToken("my_refresh_token")
	require.NoError(t, err)

	t0 := time.Now().UTC()

	tc := TokenClient{
		url:       server.URL,
		apiKey:    "my-api-key",
		expiresAt: t0,
	}

	token, err := tc.GetToken()
	require.NoError(t, err)
	require.Equal(t, Token{Type: BearerToken, Value: "my_id_token_0"}, token, "first token")

	tc.expiresAt = t0

	token, err = tc.GetToken()
	require.NoError(t, err)
	require.Equal(t, Token{Type: BearerToken, Value: "my_id_token_1"}, token, "expected to issue new token")
}

func TestTokenClient_GetToken_LongExpiry(t *testing.T) {
	server, closer := fakeAuthServer(t, "3600")
	defer closer()

	err := SaveRefreshToken("my_refresh_token")
	require.NoError(t, err)

	tc := TokenClient{
		url:    server.URL,
		apiKey: "my-api-key",
	}

	token, err := tc.GetToken()
	require.NoError(t, err)
	require.Equal(t, Token{Type: BearerToken, Value: "my_id_token_0"}, token, "first token")

	token, err = tc.GetToken()
	require.NoError(t, err)
	require.Equal(t, Token{Type: BearerToken, Value: "my_id_token_0"}, token, "expected to reuse token")
}

func TestTokenClient_BearerTokenType(t *testing.T) {
	tc := NewTokenClient()

	assert.Equal(t, BearerToken, tc.GetTokenType())
}

func TestTokenClient_APIKeyTokenType(t *testing.T) {
	t.Setenv(EnvVarCloudQueryAPIKey, "my_token")

	tc := NewTokenClient()

	assert.Equal(t, APIKey, tc.GetTokenType())
}

func TestTokenClient_SyncRunAPIKeyTokenType(t *testing.T) {
	t.Setenv(EnvVarCloudQueryAPIKey, "cqsr_my_token")

	tc := NewTokenClient()

	assert.Equal(t, SyncRunAPIKey, tc.GetTokenType())
}

func overrideEnvironmentVariable(t *testing.T, key, value string) func() {
	originalValue := os.Getenv(key)
	resetFn := func() {
		err := os.Setenv(key, originalValue)
		require.NoError(t, err)
	}

	err := os.Setenv(key, value)
	require.NoError(t, err)

	return resetFn
}

func fakeAuthServer(t *testing.T, expiresIn string) (*httptest.Server, func()) {
	tokenCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/token?key=my-api-key", r.URL.String())

		err := r.ParseForm()
		require.NoError(t, err)

		require.Equal(t, "my_refresh_token", r.Form.Get("refresh_token"))
		require.Equal(t, "refresh_token", r.Form.Get("grant_type"))

		w.Header().Set("Content-Type", "application/json")
		response := tokenResponse{
			AccessToken:  "my_access_token",
			ExpiresIn:    expiresIn,
			TokenType:    "Bearer",
			RefreshToken: "my_refresh_token",
			IDToken:      fmt.Sprintf("my_id_token_%d", tokenCount),
			UserID:       "abcd-1234",
			ProjectID:    "project-1",
		}
		err = json.NewEncoder(w).Encode(response)
		require.NoError(t, err)

		tokenCount++
	}))

	return server, func() {
		server.Close()
	}
}
