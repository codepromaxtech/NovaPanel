package config

import (
	"os"
	"time"
)

type Config struct {
	Env            string
	APIPort        string
	APIHost        string
	DBHost         string
	DBPort         string
	DBName         string
	DBUser         string
	DBPassword     string
	DBSSLMode      string
	RedisURL       string
	JWTSecret      string
	JWTExpiry      time.Duration
	AutomationURL  string
	CORSOrigins    string
	EncryptionKey  string
	FrontendURL    string
	IPWhitelist    string

	// SMTP
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string

	// Stripe
	StripeSecretKey        string
	StripeWebhookSecret    string
	StripePriceEnterprise  string
	StripePriceReseller    string

	// License
	LicenseKey        string
	LicenseServerURL  string
	LicenseProductID  string
}

func Load() *Config {
	return &Config{
		Env:           getEnv("ENV", "development"),
		APIPort:       getEnv("API_PORT", "8080"),
		APIHost:       getEnv("API_HOST", "0.0.0.0"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBName:        getEnv("DB_NAME", "novapanel"),
		DBUser:        getEnv("DB_USER", "novapanel"),
		DBPassword:    getEnv("DB_PASSWORD", "novapanel_secret"),
		DBSSLMode:     getEnv("DB_SSL_MODE", "disable"),
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6379"),
		JWTSecret:     getEnv("JWT_SECRET", "novapanel-dev-secret-change-in-production"),
		JWTExpiry:     parseDuration(getEnv("JWT_EXPIRY", "24h")),
		AutomationURL: getEnv("AUTOMATION_URL", "http://localhost:8001"),
		CORSOrigins:   getEnv("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173"),
		EncryptionKey: getEnv("ENCRYPTION_KEY", "novapanel-aes-key-change-in-prod!"),
		FrontendURL:   getEnv("FRONTEND_URL", "http://localhost:3000"),
		IPWhitelist:   getEnv("IP_WHITELIST", ""),

		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", "noreply@novapanel.io"),

		StripeSecretKey:       getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret:   getEnv("STRIPE_WEBHOOK_SECRET", ""),
		StripePriceEnterprise: getEnv("STRIPE_PRICE_ENTERPRISE", ""),
		StripePriceReseller:   getEnv("STRIPE_PRICE_RESELLER", ""),

		LicenseKey:       getEnv("LICENSE_KEY", ""),
		LicenseServerURL: getEnv("LICENSE_SERVER_URL", "https://license.codepromax.org"),
		LicenseProductID: getEnv("LICENSE_PRODUCT_ID", "novapanel"),
	}
}

func (c *Config) DatabaseDSN() string {
	return "host=" + c.DBHost +
		" port=" + c.DBPort +
		" user=" + c.DBUser +
		" password=" + c.DBPassword +
		" dbname=" + c.DBName +
		" sslmode=" + c.DBSSLMode
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 24 * time.Hour
	}
	return d
}
