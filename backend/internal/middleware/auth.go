package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/novapanel/novapanel/internal/config"
)

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		var tokenStr string

		if authHeader != "" {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == authHeader {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Bearer token required"})
				return
			}
		} else if qToken := c.Query("token"); qToken != "" {
			// WebSocket connections pass token as query parameter
			tokenStr = qToken
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			return
		}

		// Parse UserID to uuid.UUID so handlers can assert directly
		userUUID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token user_id"})
			return
		}
		c.Set("user_id", userUUID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Set("user_role", claims.Role) // alias for handlers that use "user_role"
		c.Next()
	}
}
