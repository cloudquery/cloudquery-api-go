package auth_test

import (
	"github.com/cloudquery/cloudquery-api-go/auth"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRefreshToken_RoundTrip(t *testing.T) {
	token := "my_token"

	err := auth.SaveRefreshToken(token)
	require.NoError(t, err)

	readToken, err := auth.ReadRefreshToken()
	require.NoError(t, err)

	require.Equal(t, token, readToken)
}

func TestRefreshToken_Removal(t *testing.T) {
	token := "my_token"

	err := auth.SaveRefreshToken(token)
	require.NoError(t, err)

	_, err = auth.ReadRefreshToken()
	require.NoError(t, err)

	err = auth.RemoveRefreshToken()
	require.NoError(t, err)

	_, err = auth.ReadRefreshToken()
	require.Error(t, err)
}
