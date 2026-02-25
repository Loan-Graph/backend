package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port              string
	Env               string
	DatabaseURL       string
	DBMaxConns        int32
	DBMinConns        int32
	DBMaxConnLifetime string
}

func Load() Config {
	return Config{
		Port:              getEnv("PORT", "8090"),
		Env:               getEnv("APP_ENV", "local"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://loangraph:secret@localhost:5432/loangraph?sslmode=disable"),
		DBMaxConns:        getEnvInt32("DB_MAX_CONNS", 25),
		DBMinConns:        getEnvInt32("DB_MIN_CONNS", 2),
		DBMaxConnLifetime: getEnv("DB_MAX_CONN_LIFETIME", "30m"),
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
