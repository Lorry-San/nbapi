package middleware

import (
	"github.com/Lorry-San/nbapi/common"
	"github.com/gin-gonic/gin"
)

func ResolveClientIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		common.GetClientIP(c)
		c.Next()
	}
}
