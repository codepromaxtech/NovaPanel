package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/novapanel/novapanel/internal/services"
)

// APIKeyAuth is a fallback authenticator that runs when no JWT Bearer token is present.
// It checks the X-API-Key header, validates it, and sets the same context keys as AuthMiddleware.
func APIKeyAuth(apiKeySvc *services.APIKeyService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only activate if JWT middleware hasn't already authenticated the request
		if _, exists := c.Get("user_id"); exists {
			c.Next()
			return
		}

		rawKey := c.GetHeader("X-API-Key")
		if rawKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			return
		}

		user, err := apiKeySvc.ValidateKey(c.Request.Context(), rawKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		c.Set("user_id", user.ID)
		c.Set("email", user.Email)
		c.Set("role", user.Role)
		c.Set("user_role", user.Role)
		c.Set("token_id", "") // no JTI for API key auth
		c.Next()
	}
}
