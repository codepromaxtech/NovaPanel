package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type cachedPlan struct {
	expiresAt time.Time
	// feature flags
	AllowWAF         bool
	AllowFirewall    bool
	AllowCloudflare  bool
	AllowTeam        bool
	AllowAPIKeys     bool
	AllowK8s         bool
	AllowDocker      bool
	AllowWildcardSSL bool
	AllowFTP         bool
	AllowReseller    bool
	AllowMultiDeploy bool
	// quotas
	MaxServers   int
	MaxDomains   int
	MaxDatabases int
	MaxEmail     int
}

var planCache sync.Map // map[string]*cachedPlan keyed by userID string

func loadPlan(ctx context.Context, pool *pgxpool.Pool, userID string) (*cachedPlan, error) {
	if v, ok := planCache.Load(userID); ok {
		cp := v.(*cachedPlan)
		if time.Now().Before(cp.expiresAt) {
			return cp, nil
		}
	}

	cp := &cachedPlan{expiresAt: time.Now().Add(60 * time.Second)}
	err := pool.QueryRow(ctx, `
		SELECT
			COALESCE(bp.allow_waf, false),
			COALESCE(bp.allow_firewall, false),
			COALESCE(bp.allow_cloudflare, false),
			COALESCE(bp.allow_team, false),
			COALESCE(bp.allow_api_keys, false),
			COALESCE(bp.allow_k8s, false),
			COALESCE(bp.allow_docker, false),
			COALESCE(bp.allow_wildcard_ssl, false),
			COALESCE(bp.allow_ftp, false),
			COALESCE(bp.allow_reseller, false),
			COALESCE(bp.allow_multi_deploy, false),
			COALESCE(bp.max_servers, 1),
			COALESCE(bp.max_domains, 3),
			COALESCE(bp.max_databases, 2),
			COALESCE(bp.max_email, 10)
		FROM users u
		LEFT JOIN billing_plans bp ON bp.id = u.plan_id
		WHERE u.id = $1`, userID,
	).Scan(
		&cp.AllowWAF, &cp.AllowFirewall, &cp.AllowCloudflare, &cp.AllowTeam,
		&cp.AllowAPIKeys, &cp.AllowK8s, &cp.AllowDocker, &cp.AllowWildcardSSL,
		&cp.AllowFTP, &cp.AllowReseller, &cp.AllowMultiDeploy,
		&cp.MaxServers, &cp.MaxDomains, &cp.MaxDatabases, &cp.MaxEmail,
	)
	if err != nil {
		// Default to community limits on error
		cp.MaxServers = 1
		cp.MaxDomains = 3
		cp.MaxDatabases = 2
		cp.MaxEmail = 10
	}
	planCache.Store(userID, cp)
	return cp, nil
}

// FeatureGate blocks access to enterprise-only features.
// feature must be one of: "allow_waf", "allow_firewall", "allow_cloudflare",
// "allow_team", "allow_api_keys", "allow_k8s", "allow_docker",
// "allow_wildcard_ssl", "allow_ftp", "allow_reseller", "allow_multi_deploy"
func FeatureGate(pool *pgxpool.Pool, feature string) gin.HandlerFunc {
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
		uid := userID.(uuid.UUID).String()
		plan, err := loadPlan(c.Request.Context(), pool, uid)
		if err != nil {
			c.Next()
			return
		}

		allowed := false
		switch feature {
		case "allow_waf":
			allowed = plan.AllowWAF
		case "allow_firewall":
			allowed = plan.AllowFirewall
		case "allow_cloudflare":
			allowed = plan.AllowCloudflare
		case "allow_team":
			allowed = plan.AllowTeam
		case "allow_api_keys":
			allowed = plan.AllowAPIKeys
		case "allow_k8s":
			allowed = plan.AllowK8s
		case "allow_docker":
			allowed = plan.AllowDocker
		case "allow_wildcard_ssl":
			allowed = plan.AllowWildcardSSL
		case "allow_ftp":
			allowed = plan.AllowFTP
		case "allow_reseller":
			allowed = plan.AllowReseller
		case "allow_multi_deploy":
			allowed = plan.AllowMultiDeploy
		default:
			allowed = true
		}

		if !allowed {
			c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
				"error":   "upgrade_required",
				"feature": feature,
				"message": "This feature requires an Enterprise or higher plan",
			})
			return
		}
		c.Next()
	}
}

// ResourceQuota blocks resource creation when the user is at their plan limit.
// resource must be one of: "domains", "databases", "apps", "servers", "email"
func ResourceQuota(pool *pgxpool.Pool, resource string) gin.HandlerFunc {
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
		uid := userID.(uuid.UUID).String()
		plan, err := loadPlan(c.Request.Context(), pool, uid)
		if err != nil {
			c.Next()
			return
		}

		var count int
		var limit int
		var tableName string

		switch resource {
		case "domains":
			tableName = "domains"
			limit = plan.MaxDomains
		case "databases":
			tableName = "databases"
			limit = plan.MaxDatabases
		case "apps":
			tableName = "applications"
			limit = 50 // apps not directly capped by plan columns — generous default
		case "servers":
			tableName = "servers"
			limit = plan.MaxServers
		case "email":
			tableName = "email_accounts"
			limit = plan.MaxEmail
		default:
			c.Next()
			return
		}

		if tableName == "servers" {
			// servers are global to the account, not per user_id
			err = pool.QueryRow(c.Request.Context(),
				"SELECT COUNT(*) FROM "+tableName+" WHERE status != 'deleted'",
			).Scan(&count)
		} else {
			err = pool.QueryRow(c.Request.Context(),
				"SELECT COUNT(*) FROM "+tableName+" WHERE user_id = $1", uid,
			).Scan(&count)
		}

		if err == nil && count >= limit {
			c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
				"error":   "quota_exceeded",
				"resource": resource,
				"limit":   limit,
				"current": count,
				"message": "You have reached your plan limit. Please upgrade to add more resources.",
			})
			return
		}
		c.Next()
	}
}

// InvalidatePlanCache removes a user's cached plan entry (call after plan change).
func InvalidatePlanCache(userID string) {
	planCache.Delete(userID)
}
