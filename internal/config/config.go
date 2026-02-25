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

	WorkerPollInterval     time.Duration
	WorkerBatchSize        int32
	ChainWriterMode        string
	CreditcoinHTTPRPC      string
	CreditcoinChainID      int64
	LoanRegistryProxy      string
	ChainWriterFromAddress string
	LenderSignerPrivateKey string
	ChainTxGasLimit        uint64
	IndexerPollInterval    time.Duration
	IndexerBatchSize       int32
	WSEnabled              bool
	WSPollInterval         time.Duration
	MaxRequestBodyBytes    int64
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

		WorkerPollInterval:     getEnvDuration("WORKER_POLL_INTERVAL", 2*time.Second),
		WorkerBatchSize:        getEnvInt32("WORKER_BATCH_SIZE", 20),
		ChainWriterMode:        getEnv("CHAIN_WRITER_MODE", "stub"),
		CreditcoinHTTPRPC:      getEnv("CREDITCOIN_HTTP_RPC", ""),
		CreditcoinChainID:      getEnvInt64("CREDITCOIN_CHAIN_ID", 102031),
		LoanRegistryProxy:      getEnv("LOAN_REGISTRY_PROXY", ""),
		ChainWriterFromAddress: getEnv("CHAIN_WRITER_FROM_ADDRESS", ""),
		LenderSignerPrivateKey: getEnv("LENDER_SIGNER_PRIVATE_KEY", ""),
		ChainTxGasLimit:        getEnvUint64("CHAIN_TX_GAS_LIMIT", 300000),
		IndexerPollInterval:    getEnvDuration("INDEXER_POLL_INTERVAL", 2*time.Second),
		IndexerBatchSize:       getEnvInt32("INDEXER_BATCH_SIZE", 100),
		WSEnabled:              getEnvBool("WS_ENABLED", true),
		WSPollInterval:         getEnvDuration("WS_POLL_INTERVAL", 2*time.Second),
		MaxRequestBodyBytes:    getEnvInt64("MAX_REQUEST_BODY_BYTES", 62914560), // 60 MiB
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

func getEnvInt64(key string, fallback int64) int64 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		var out int64
		_, err := fmt.Sscanf(v, "%d", &out)
		if err == nil {
			return out
		}
	}
	return fallback
}

func getEnvUint64(key string, fallback uint64) uint64 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		var out uint64
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
