package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testClientIP(t *testing.T, enabled bool, remoteAddr string, headers map[string]string) string {
	t.Helper()
	previous := IsTrustedProxyValidationEnabled()
	SetTrustedProxyValidationEnabled(enabled)
	t.Cleanup(func() {
		SetTrustedProxyValidationEnabled(previous)
	})

	router := gin.New()
	require.NoError(t, router.SetTrustedProxies([]string{"127.0.0.0/8"}))
	router.GET("/client-ip", func(c *gin.Context) {
		c.String(http.StatusOK, GetClientIP(c))
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/client-ip", nil)
	request.RemoteAddr = remoteAddr
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	router.ServeHTTP(recorder, request)
	return recorder.Body.String()
}

func TestGetClientIPUsesTrustedProxyHeadersWhenEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assert.Equal(t, "203.0.113.10", testClientIP(t, true, "127.0.0.1:12345", map[string]string{
		"X-Forwarded-For": "203.0.113.10",
	}))
}

func TestGetClientIPRejectsUntrustedProxyHeadersWhenValidationEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assert.Equal(t, "198.51.100.20", testClientIP(t, true, "198.51.100.20:12345", map[string]string{
		"X-Forwarded-For": "203.0.113.10",
	}))
}

func TestGetClientIPTrustsForwardedHeadersWhenValidationDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assert.Equal(t, "203.0.113.10", testClientIP(t, false, "198.51.100.20:12345", map[string]string{
		"X-Forwarded-For": "203.0.113.10, 198.51.100.30",
	}))
}

func TestGetClientIPPrefersCloudflareHeaderWhenValidationDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assert.Equal(t, "203.0.113.11", testClientIP(t, false, "198.51.100.20:12345", map[string]string{
		"CF-Connecting-IP": "203.0.113.11",
		"X-Forwarded-For":  "203.0.113.12",
	}))
}

func TestGetClientIPFallsBackToDirectAddressWithoutValidHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assert.Equal(t, "198.51.100.20", testClientIP(t, false, "198.51.100.20:12345", map[string]string{
		"X-Forwarded-For": "unknown, invalid",
	}))
}

func TestDirectClientIPSupportsIPv6(t *testing.T) {
	assert.Equal(t, "2001:db8::1", directClientIP("[2001:db8::1]:443"))
}
