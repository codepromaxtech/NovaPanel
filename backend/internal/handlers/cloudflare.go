package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/services"
)

type CloudflareHandler struct {
	svc *services.CloudflareService
}

func NewCloudflareHandler(svc *services.CloudflareService) *CloudflareHandler {
	return &CloudflareHandler{svc: svc}
}

// helper to extract CF creds from request
type cfAuth struct {
	APIKey string `json:"api_key" binding:"required"`
	Email  string `json:"email"`
}

func (h *CloudflareHandler) proxy(c *gin.Context, fn func(auth cfAuth, body map[string]interface{}) (map[string]interface{}, error)) {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	apiKey, _ := body["api_key"].(string)
	email, _ := body["email"].(string)
	if apiKey == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "api_key is required"})
		return
	}
	auth := cfAuth{APIKey: apiKey, Email: email}
	result, err := fn(auth, body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}

// POST /cloudflare/verify
func (h *CloudflareHandler) Verify(c *gin.Context) {
	h.proxy(c, func(auth cfAuth, body map[string]interface{}) (map[string]interface{}, error) {
		return h.svc.VerifyToken(c.Request.Context(), auth.APIKey, auth.Email)
	})
}

// POST /cloudflare/zones
func (h *CloudflareHandler) ListZones(c *gin.Context) {
	h.proxy(c, func(auth cfAuth, body map[string]interface{}) (map[string]interface{}, error) {
		return h.svc.ListZones(c.Request.Context(), auth.APIKey, auth.Email, getInt(body, "page"))
	})
}

// POST /cloudflare/zones/get
func (h *CloudflareHandler) GetZone(c *gin.Context) {
	h.proxy(c, func(auth cfAuth, body map[string]interface{}) (map[string]interface{}, error) {
		return h.svc.GetZone(c.Request.Context(), auth.APIKey, auth.Email, getString(body, "zone_id"))
	})
}

// POST /cloudflare/dns/list
func (h *CloudflareHandler) ListDNS(c *gin.Context) {
	h.proxy(c, func(auth cfAuth, body map[string]interface{}) (map[string]interface{}, error) {
		return h.svc.ListDNSRecords(c.Request.Context(), auth.APIKey, auth.Email, getString(body, "zone_id"), getInt(body, "page"))
	})
}

// POST /cloudflare/dns/create
func (h *CloudflareHandler) CreateDNS(c *gin.Context) {
	h.proxy(c, func(auth cfAuth, body map[string]interface{}) (map[string]interface{}, error) {
		record := map[string]interface{}{
			"type":    getString(body, "type"),
			"name":    getString(body, "name"),
			"content": getString(body, "content"),
			"ttl":     getInt(body, "ttl"),
		}
		if v, ok := body["proxied"]; ok {
			record["proxied"] = v
		}
		if p := getInt(body, "priority"); p > 0 {
			record["priority"] = p
		}
		return h.svc.CreateDNSRecord(c.Request.Context(), auth.APIKey, auth.Email, getString(body, "zone_id"), record)
	})
}

// POST /cloudflare/dns/update
func (h *CloudflareHandler) UpdateDNS(c *gin.Context) {
	h.proxy(c, func(auth cfAuth, body map[string]interface{}) (map[string]interface{}, error) {
		record := map[string]interface{}{
			"type":    getString(body, "type"),
			"name":    getString(body, "name"),
			"content": getString(body, "content"),
			"ttl":     getInt(body, "ttl"),
		}
		if v, ok := body["proxied"]; ok {
			record["proxied"] = v
		}
		return h.svc.UpdateDNSRecord(c.Request.Context(), auth.APIKey, auth.Email, getString(body, "zone_id"), getString(body, "record_id"), record)
	})
}

// POST /cloudflare/dns/delete
func (h *CloudflareHandler) DeleteDNS(c *gin.Context) {
	h.proxy(c, func(auth cfAuth, body map[string]interface{}) (map[string]interface{}, error) {
		return h.svc.DeleteDNSRecord(c.Request.Context(), auth.APIKey, auth.Email, getString(body, "zone_id"), getString(body, "record_id"))
	})
}

// POST /cloudflare/ssl/get
func (h *CloudflareHandler) GetSSL(c *gin.Context) {
	h.proxy(c, func(auth cfAuth, body map[string]interface{}) (map[string]interface{}, error) {
		return h.svc.GetSSLSetting(c.Request.Context(), auth.APIKey, auth.Email, getString(body, "zone_id"))
	})
}

// POST /cloudflare/ssl/set
func (h *CloudflareHandler) SetSSL(c *gin.Context) {
	h.proxy(c, func(auth cfAuth, body map[string]interface{}) (map[string]interface{}, error) {
		return h.svc.SetSSLSetting(c.Request.Context(), auth.APIKey, auth.Email, getString(body, "zone_id"), getString(body, "mode"))
	})
}

// POST /cloudflare/cache/purge-all
func (h *CloudflareHandler) PurgeAll(c *gin.Context) {
	h.proxy(c, func(auth cfAuth, body map[string]interface{}) (map[string]interface{}, error) {
		return h.svc.PurgeAllCache(c.Request.Context(), auth.APIKey, auth.Email, getString(body, "zone_id"))
	})
}

// POST /cloudflare/cache/purge-urls
func (h *CloudflareHandler) PurgeURLs(c *gin.Context) {
	h.proxy(c, func(auth cfAuth, body map[string]interface{}) (map[string]interface{}, error) {
		urlsRaw, _ := body["urls"].([]interface{})
		urls := make([]string, len(urlsRaw))
		for i, u := range urlsRaw {
			urls[i], _ = u.(string)
		}
		return h.svc.PurgeURLs(c.Request.Context(), auth.APIKey, auth.Email, getString(body, "zone_id"), urls)
	})
}

// POST /cloudflare/cache/ttl
func (h *CloudflareHandler) SetCacheTTL(c *gin.Context) {
	h.proxy(c, func(auth cfAuth, body map[string]interface{}) (map[string]interface{}, error) {
		return h.svc.SetCacheTTL(c.Request.Context(), auth.APIKey, auth.Email, getString(body, "zone_id"), getInt(body, "ttl"))
	})
}

// POST /cloudflare/devmode
func (h *CloudflareHandler) SetDevMode(c *gin.Context) {
	h.proxy(c, func(auth cfAuth, body map[string]interface{}) (map[string]interface{}, error) {
		return h.svc.SetDevMode(c.Request.Context(), auth.APIKey, auth.Email, getString(body, "zone_id"), getString(body, "value"))
	})
}

// POST /cloudflare/security
func (h *CloudflareHandler) SetSecurity(c *gin.Context) {
	h.proxy(c, func(auth cfAuth, body map[string]interface{}) (map[string]interface{}, error) {
		return h.svc.SetSecurityLevel(c.Request.Context(), auth.APIKey, auth.Email, getString(body, "zone_id"), getString(body, "level"))
	})
}

// POST /cloudflare/firewall/list
func (h *CloudflareHandler) ListFirewall(c *gin.Context) {
	h.proxy(c, func(auth cfAuth, body map[string]interface{}) (map[string]interface{}, error) {
		return h.svc.ListFirewallRules(c.Request.Context(), auth.APIKey, auth.Email, getString(body, "zone_id"))
	})
}

// POST /cloudflare/analytics
func (h *CloudflareHandler) Analytics(c *gin.Context) {
	h.proxy(c, func(auth cfAuth, body map[string]interface{}) (map[string]interface{}, error) {
		return h.svc.GetAnalytics(c.Request.Context(), auth.APIKey, auth.Email, getString(body, "zone_id"), getInt(body, "since"))
	})
}

// POST /cloudflare/settings/get
func (h *CloudflareHandler) GetSettings(c *gin.Context) {
	h.proxy(c, func(auth cfAuth, body map[string]interface{}) (map[string]interface{}, error) {
		zoneID := getString(body, "zone_id")
		ctx := c.Request.Context()
		ssl, _ := h.svc.GetSSLSetting(ctx, auth.APIKey, auth.Email, zoneID)
		devMode, _ := h.svc.GetDevMode(ctx, auth.APIKey, auth.Email, zoneID)
		secLevel, _ := h.svc.GetSecurityLevel(ctx, auth.APIKey, auth.Email, zoneID)
		alwaysHTTPS, _ := h.svc.GetAlwaysHTTPS(ctx, auth.APIKey, auth.Email, zoneID)
		rocketLoader, _ := h.svc.GetRocketLoader(ctx, auth.APIKey, auth.Email, zoneID)
		minify, _ := h.svc.GetMinify(ctx, auth.APIKey, auth.Email, zoneID)
		cacheTTL, _ := h.svc.GetCacheSetting(ctx, auth.APIKey, auth.Email, zoneID)
		return map[string]interface{}{
			"ssl":            ssl,
			"dev_mode":       devMode,
			"security_level": secLevel,
			"always_https":   alwaysHTTPS,
			"rocket_loader":  rocketLoader,
			"minify":         minify,
			"cache_ttl":      cacheTTL,
		}, nil
	})
}

// POST /cloudflare/settings/update
func (h *CloudflareHandler) UpdateSetting(c *gin.Context) {
	h.proxy(c, func(auth cfAuth, body map[string]interface{}) (map[string]interface{}, error) {
		zoneID := getString(body, "zone_id")
		setting := getString(body, "setting")
		value := getString(body, "value")
		ctx := c.Request.Context()
		switch setting {
		case "ssl":
			return h.svc.SetSSLSetting(ctx, auth.APIKey, auth.Email, zoneID, value)
		case "always_use_https":
			return h.svc.SetAlwaysHTTPS(ctx, auth.APIKey, auth.Email, zoneID, value)
		case "rocket_loader":
			return h.svc.SetRocketLoader(ctx, auth.APIKey, auth.Email, zoneID, value)
		case "development_mode":
			return h.svc.SetDevMode(ctx, auth.APIKey, auth.Email, zoneID, value)
		case "security_level":
			return h.svc.SetSecurityLevel(ctx, auth.APIKey, auth.Email, zoneID, value)
		default:
			return nil, fmt.Errorf("unknown setting: %s", setting)
		}
	})
}
