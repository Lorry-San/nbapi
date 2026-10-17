package service

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"

	"github.com/Lorry-San/nbapi/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const lightcodeTestRedirect = "http://127.0.0.1:49152/oauth/lightcode/callback"

func TestLightcodeOAuthRotationAndReplayRevocation(t *testing.T) {
	user := setupAuthSessionTestDB(t)
	useTestSessionSecret(t)
	require.NoError(t, model.DB.AutoMigrate(&model.LightcodeAuthCode{}, &model.LightcodeSession{}, &model.LightcodeRefreshToken{}))
	verifier := "lightcode-verifier-that-is-long-enough-for-rfc7636-0123456789"
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	code, err := AuthorizeLightcode(user.Id, LightcodeClientID, lightcodeTestRedirect, challenge, "S256")
	require.NoError(t, err)
	_, err = ExchangeLightcodeGrant(code, LightcodeClientID, lightcodeTestRedirect, verifier+"x", "", "")
	require.ErrorIs(t, err, model.ErrLightcodeGrant)
	first, err := ExchangeLightcodeGrant(code, LightcodeClientID, lightcodeTestRedirect, verifier, "127.0.0.1", "test")
	require.NoError(t, err)
	_, err = ExchangeLightcodeGrant(code, LightcodeClientID, lightcodeTestRedirect, verifier, "", "")
	require.ErrorIs(t, err, model.ErrLightcodeGrant, "authorization codes are single use")

	_, err = ValidateLightcodeAccess(first.AccessToken)
	require.NoError(t, err)
	_, err = ParseAccessToken(first.AccessToken)
	require.Error(t, err, "Lightcode access tokens must not authenticate the dashboard")

	second, err := RefreshLightcodeGrant(first.RefreshToken, LightcodeClientID)
	require.NoError(t, err)
	retry, err := RefreshLightcodeGrant(first.RefreshToken, LightcodeClientID)
	require.NoError(t, err, "a lost response may be retried inside the grace window")
	assert.Equal(t, second.RefreshToken, retry.RefreshToken)

	third, err := RefreshLightcodeGrant(second.RefreshToken, LightcodeClientID)
	require.NoError(t, err)
	_, err = RefreshLightcodeGrant(second.RefreshToken+"tampered", LightcodeClientID)
	require.ErrorIs(t, err, model.ErrLightcodeGrant)

	require.NoError(t, model.DB.Model(&model.LightcodeRefreshToken{}).Where("used_at > 0").Update("used_at", 1).Error)
	_, err = RefreshLightcodeGrant(first.RefreshToken, LightcodeClientID)
	require.ErrorIs(t, err, model.ErrLightcodeGrant, "replay outside the grace window is rejected")
	_, err = ValidateLightcodeAccess(third.AccessToken)
	require.ErrorIs(t, err, model.ErrLightcodeGrant, "replay revokes the whole device session")
}

func TestLightcodeRedirectOnlyAcceptsTheLoopbackCallback(t *testing.T) {
	for _, redirect := range []string{
		"https://127.0.0.1:49152/oauth/lightcode/callback",
		"http://evil.example:49152/oauth/lightcode/callback",
		"http://127.0.0.1/oauth/lightcode/callback",
		"http://127.0.0.1:49152/other",
		"http://user@127.0.0.1:49152/oauth/lightcode/callback",
	} {
		assert.Error(t, ValidateLightcodeRedirect(LightcodeClientID, redirect), redirect)
	}
	assert.NoError(t, ValidateLightcodeRedirect(LightcodeClientID, lightcodeTestRedirect))
	assert.Error(t, ValidateLightcodeRedirect("other", lightcodeTestRedirect))
}
