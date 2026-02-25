package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Port              string
	Env               string
	DatabaseURL       string
	DBMaxConns        int32
	DBMinConns        int32
	DBMaxConnLifetime string

	JWTIssuer     string
	JWTAudience   string
	JWTSigningKey string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	CookieDomain string
	CookieSecure bool

	PrivyAppID           string
	PrivyAppSecret       string
	PrivyJWKSURL         string
	PrivyIssuer          string
	PrivyAudience        string
	PrivyVerificationKey string

	AuthEnableBearer          bool
	AuthBootstrapAdminSubject string
}

func Load() Config {
	return Config{
		Port:              getEnv("PORT", "8090"),
		Env:               getEnv("APP_ENV", "local"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://loangraph:secret@localhost:5432/loangraph?sslmode=disable"),
		DBMaxConns:        getEnvInt32("DB_MAX_CONNS", 25),
		DBMinConns:        getEnvInt32("DB_MIN_CONNS", 2),
		DBMaxConnLifetime: getEnv("DB_MAX_CONN_LIFETIME", "30m"),

		JWTIssuer:     getEnv("JWT_ISSUER", "loangraph-backend"),
		JWTAudience:   getEnv("JWT_AUDIENCE", "loangraph-api"),
		JWTSigningKey: getEnv("JWT_SIGNING_KEY", "dev-insecure-key-change-me"),
		JWTAccessTTL:  getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL: getEnvDuration("JWT_REFRESH_TTL", 7*24*time.Hour),

		CookieDomain: getEnv("COOKIE_DOMAIN", ""),
		CookieSecure: getEnvBool("COOKIE_SECURE", false),

		PrivyAppID:           getEnv("PRIVY_APP_ID", ""),
		PrivyAppSecret:       getEnv("PRIVY_APP_SECRET", ""),
		PrivyJWKSURL:         getEnv("PRIVY_JWKS_URL", ""),
		PrivyIssuer:          getEnv("PRIVY_ISSUER", ""),
		PrivyAudience:        getEnv("PRIVY_AUDIENCE", ""),
		PrivyVerificationKey: getEnv("PRIVY_VERIFICATION_KEY", ""),

		AuthEnableBearer:          getEnvBool("AUTH_ENABLE_BEARER", false),
		AuthBootstrapAdminSubject: getEnv("AUTH_BOOTSTRAP_ADMIN_SUBJECT", ""),
	}
}

func (c Config) Addr() string {
	return fmt.Sprintf(":%s", c.Port)
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt32(key string, fallback int32) int32 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		var out int32
		_, err := fmt.Sscanf(v, "%d", &out)
		if err == nil {
			return out
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		d, err := time.ParseDuration(v)
		if err == nil {
			return d
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		n := strings.ToLower(strings.TrimSpace(v))
		return n == "1" || n == "true" || n == "yes"
	}
	return fallback
}
