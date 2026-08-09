package common

import (
	"net"
	"strings"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

const ResolvedClientIPKey = "nbapi_resolved_client_ip"

var trustedProxiesEnabled atomic.Bool

func init() {
	trustedProxiesEnabled.Store(true)
}

func IsTrustedProxiesEnabled() bool {
	return trustedProxiesEnabled.Load()
}

func SetTrustedProxiesEnabled(enabled bool) {
	trustedProxiesEnabled.Store(enabled)
}

func GetClientIP(c *gin.Context) string {
	if resolvedIP, exists := c.Get(ResolvedClientIPKey); exists {
		if clientIP, ok := resolvedIP.(string); ok && clientIP != "" {
			return clientIP
		}
	}

	clientIP := directClientIP(c.Request.RemoteAddr)
	if IsTrustedProxiesEnabled() {
		clientIP = c.ClientIP()
	}
	c.Set(ResolvedClientIPKey, clientIP)
	return clientIP
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
