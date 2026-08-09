package common

import (
	"net"
	"strings"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

const ResolvedClientIPKey = "nbapi_resolved_client_ip"

var trustedProxyValidationEnabled atomic.Bool

var unverifiedClientIPHeaders = []string{
	"CF-Connecting-IP",
	"True-Client-IP",
	"X-Forwarded-For",
	"X-Real-IP",
}

func init() {
	trustedProxyValidationEnabled.Store(true)
}

func IsTrustedProxyValidationEnabled() bool {
	return trustedProxyValidationEnabled.Load()
}

func SetTrustedProxyValidationEnabled(enabled bool) {
	trustedProxyValidationEnabled.Store(enabled)
}

func GetClientIP(c *gin.Context) string {
	if resolvedIP, exists := c.Get(ResolvedClientIPKey); exists {
		if clientIP, ok := resolvedIP.(string); ok && clientIP != "" {
			return clientIP
		}
	}

	var clientIP string
	if IsTrustedProxyValidationEnabled() {
		clientIP = c.ClientIP()
	} else {
		clientIP = forwardedClientIP(c)
		if clientIP == "" {
			clientIP = directClientIP(c.Request.RemoteAddr)
		}
	}
	c.Set(ResolvedClientIPKey, clientIP)
	return clientIP
}

func forwardedClientIP(c *gin.Context) string {
	for _, headerName := range unverifiedClientIPHeaders {
		for _, headerValue := range c.Request.Header.Values(headerName) {
			for _, candidate := range strings.Split(headerValue, ",") {
				if clientIP := normalizedIP(candidate); clientIP != "" {
					return clientIP
				}
			}
		}
	}
	return ""
}

func normalizedIP(value string) string {
	value = strings.Trim(strings.TrimSpace(value), "\"")
	if ip := net.ParseIP(value); ip != nil {
		return ip.String()
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		if ip := net.ParseIP(strings.Trim(host, "[]")); ip != nil {
			return ip.String()
		}
	}
	return ""
}

func directClientIP(remoteAddr string) string {
	remoteAddr = strings.TrimSpace(remoteAddr)
	if ip := net.ParseIP(remoteAddr); ip != nil {
		return ip.String()
	}
	if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
		return strings.Trim(host, "[]")
	}
	return strings.Trim(remoteAddr, "[]")
}
