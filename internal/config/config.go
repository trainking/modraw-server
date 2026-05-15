package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port         string
	DatabaseURL  string
	JWTSecret    string
	AccessTTL    time.Duration
	RefreshTTL   time.Duration
	CORSOrigins  []string
	WSMaxMsgSize int64
	GinMode      string
}

func Load() *Config {
	return &Config{
		Port:         getEnv("PORT", ":8080"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://modraw:modraw@localhost:5432/modraw?sslmode=disable"),
		JWTSecret:    getEnv("JWT_SECRET", "change-me-in-production"),
		AccessTTL:    getDurationEnv("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTTL:   getDurationEnv("REFRESH_TOKEN_TTL", 7*24*time.Hour),
		CORSOrigins:  getSliceEnv("CORS_ORIGINS", []string{"*"}),
		WSMaxMsgSize: getInt64Env("WS_MAX_MSG_SIZE", 4096),
		GinMode:      getEnv("GIN_MODE", "debug"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return fallback
}

func getInt64Env(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			return n
		}
	}
	return fallback
}

func getSliceEnv(key string, fallback []string) []string {
	if v := os.Getenv(key); v != "" {
		parts := strings.Split(v, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				result = append(result, p)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return fallback
}
