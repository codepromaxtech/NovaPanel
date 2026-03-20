package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RequireRole creates middleware that restricts access to specific roles.
// Roles are hierarchical: admin > reseller > client
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		role := userRole.(string)
		for _, allowed := range roles {
			if role == allowed {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "Insufficient permissions. Required role: " + roles[0],
		})
	}
}

// RequireAdmin is a shortcut for admin-only routes
func RequireAdmin() gin.HandlerFunc {
	return RequireRole("admin")
}

// RequireReseller allows admin and reseller access
func RequireReseller() gin.HandlerFunc {
	return RequireRole("admin", "reseller")
}
