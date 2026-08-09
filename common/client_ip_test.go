package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testClientIP(t *testing.T, enabled bool, remoteAddr string, forwardedFor string) string {
	t.Helper()
	previous := IsTrustedProxiesEnabled()
	SetTrustedProxiesEnabled(enabled)
	t.Cleanup(func() {
		SetTrustedProxiesEnabled(previous)
	})

	router := gin.New()
	require.NoError(t, router.SetTrustedProxies([]string{"127.0.0.0/8"}))
	router.GET("/client-ip", func(c *gin.Context) {
		c.String(http.StatusOK, GetClientIP(c))
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/client-ip", nil)
	request.RemoteAddr = remoteAddr
	request.Header.Set("X-Forwarded-For", forwardedFor)
	router.ServeHTTP(recorder, request)
	return recorder.Body.String()
}

func TestGetClientIPUsesTrustedProxyHeadersWhenEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assert.Equal(t, "203.0.113.10", testClientIP(t, true, "127.0.0.1:12345", "203.0.113.10"))
}

func TestGetClientIPIgnoresTrustedProxyHeadersWhenDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	assert.Equal(t, "127.0.0.1", testClientIP(t, false, "127.0.0.1:12345", "203.0.113.10"))
}

func TestDirectClientIPSupportsIPv6(t *testing.T) {
	assert.Equal(t, "2001:db8::1", directClientIP("[2001:db8::1]:443"))
}
