package middleware

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/novapanel/novapanel/internal/config"
)

func CORSMiddleware() gin.HandlerFunc {
	cfg := config.Load()
	origins := strings.Split(cfg.CORSOrigins, ",")
	for i, o := range origins {
		origins[i] = strings.TrimSpace(o)
	}
	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}
