package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/ziwenx1973/GoArena/internal/service"
	"strings"
)

func JWT(auth *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := strings.Fields(c.GetHeader("Authorization"))
		if len(p) != 2 || !strings.EqualFold(p[0], "Bearer") {
			c.AbortWithStatusJSON(401, gin.H{"error": "Bearer token required"})
			return
		}
		claims, err := auth.Parse(p[1])
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
