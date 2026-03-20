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
