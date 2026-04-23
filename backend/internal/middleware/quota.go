package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/services"
)

// licenseProvider is a thin interface so we don't import the full services package in a circular way.
type licenseProvider interface {
	GetStatus() services.LicenseStatus
}

// FeatureGate blocks access to enterprise-only features.
// The check is against the installation license (not per-user plans).
// Admin users always bypass the gate.
//
// feature must be one of:
//
//	"allow_waf", "allow_firewall", "allow_cloudflare", "allow_team",
//	"allow_api_keys", "allow_k8s", "allow_docker", "allow_wildcard_ssl",
//	"allow_ftp", "allow_reseller", "allow_multi_deploy"
func FeatureGate(lsvc licenseProvider, feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("user_role")
		if role == "admin" {
			c.Next()
			return
		}

		status := lsvc.GetStatus()
		f := status.Features

		allowed := false
		switch feature {
		case "allow_waf":
			allowed = f.AllowWAF
		case "allow_firewall":
			allowed = f.AllowFirewall
		case "allow_cloudflare":
			allowed = f.AllowCloudflare
		case "allow_team":
			allowed = f.AllowTeam
		case "allow_api_keys":
			allowed = f.AllowAPIKeys
		case "allow_k8s":
			allowed = f.AllowK8s
		case "allow_docker":
			allowed = f.AllowDocker
		case "allow_wildcard_ssl":
			allowed = f.AllowWildcardSSL
		case "allow_ftp":
			allowed = f.AllowFTP
		case "allow_reseller":
			allowed = f.AllowReseller
		case "allow_multi_deploy":
			allowed = f.AllowMultiDeploy
		default:
			allowed = true
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
				"error":    "upgrade_required",
				"feature":  feature,
				"plan":     status.PlanType,
				"message":  "This feature is not available on your current plan. Please upgrade.",
			})
			return
		}
		c.Next()
	}
}

// ResourceQuota blocks resource creation when the user is at the installation plan limit.
// resource must be one of: "domains", "databases", "apps", "servers", "email"
func ResourceQuota(pool *pgxpool.Pool, lsvc licenseProvider, resource string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}
		role, _ := c.Get("user_role")
		if role == "admin" {
			c.Next()
			return
		}

		status := lsvc.GetStatus()
		f := status.Features

		var count int
		var limit int
		var tableName string

		switch resource {
		case "domains":
			tableName = "domains"
			limit = f.MaxDomains
		case "databases":
			tableName = "databases"
			limit = f.MaxDatabases
		case "apps":
			tableName = "applications"
			limit = 50
		case "servers":
			tableName = "servers"
			limit = f.MaxServers
		case "email":
			tableName = "email_accounts"
			limit = f.MaxEmail
		default:
			c.Next()
			return
		}

		uid := userID.(uuid.UUID).String()
		var err error

		if tableName == "servers" {
			err = pool.QueryRow(c.Request.Context(),
				"SELECT COUNT(*) FROM "+tableName+" WHERE status != 'deleted'",
			).Scan(&count)
		} else {
			err = pool.QueryRow(c.Request.Context(),
				"SELECT COUNT(*) FROM "+tableName+" WHERE user_id = $1", uid,
			).Scan(&count)
		}

		if err == nil && limit > 0 && count >= limit {
			c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
				"error":    "quota_exceeded",
				"resource": resource,
				"limit":    limit,
				"current":  count,
				"plan":     status.PlanType,
				"message":  "You have reached your plan limit. Please upgrade.",
			})
			return
		}
		c.Next()
	}
}

// InvalidatePlanCache is kept for compatibility but is a no-op now
// (license is refreshed on a timer, not per-request).
func InvalidatePlanCache(_ string) {}
