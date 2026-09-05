package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Env            string
	Port           int
	AllowedOrigins []string
	LoggerLevel    string
	GatewayToken   string
	SwaggerEnabled bool
	JWT            JWTConfig
	Postgres       PostgresConfig
	Redis          RedisConfig
	OAuth          OAuthConfig
	Gateway        GatewayConfig
}

type JWTConfig struct {
	Secret string
	TTL    time.Duration
}

type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	TenantID     string
	Issuer       string
	RedirectURL  string
	FrontendURL  string
	DefaultRole  string
}

// GatewayConfig wires the registered-API registry to the Nginx gateway.
// ConfigDir is the api-side mount of the shared registry volume; when empty
// the registry still works but nothing is published to a gateway.
type GatewayConfig struct {
	ConfigDir    string
	AuthUpstream string
}

func Load() *Config {
	return &Config{
		Env:            getString("APP_ENV", "development"),
		Port:           getInt("APP_PORT", 8080),
		AllowedOrigins: splitComma(getString("APP_ALLOWED_ORIGINS", "*")),
		LoggerLevel:    getString("APP_LOGGER_LEVEL", "debug"),
		GatewayToken:   getString("APP_GATEWAY_TOKEN", ""),
		SwaggerEnabled: getBool("APP_SWAGGER_ENABLED", true),
		JWT: JWTConfig{
			Secret: getString("APP_JWT_SECRET", "change-me-in-production"),
			TTL:    getDuration("APP_JWT_TTL", 15*time.Minute),
		},
		Postgres: PostgresConfig{
			Host:     getString("POSTGRES_HOST", "localhost"),
			Port:     getInt("POSTGRES_PORT", 5432),
			User:     getString("POSTGRES_USER", "app"),
			Password: getString("POSTGRES_PASSWORD", "app"),
			DBName:   getString("POSTGRES_DB", "go_enterprise"),
			SSLMode:  getString("POSTGRES_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getString("REDIS_HOST", "localhost"),
			Port:     getInt("REDIS_PORT", 6379),
			Password: getString("REDIS_PASSWORD", ""),
			DB:       getInt("REDIS_DB", 0),
		},
		OAuth: OAuthConfig{
			ClientID:     getString("OAUTH_CLIENT_ID", ""),
			ClientSecret: getString("OAUTH_CLIENT_SECRET", ""),
			TenantID:     getString("OAUTH_TENANT_ID", ""),
			Issuer:       getString("OAUTH_ISSUER", ""),
			RedirectURL:  getString("OAUTH_REDIRECT_URL", ""),
			FrontendURL:  getString("OAUTH_FRONTEND_URL", ""),
			DefaultRole:  getString("OAUTH_DEFAULT_ROLE", "viewer"),
		},
		Gateway: GatewayConfig{
			ConfigDir:    getString("GATEWAY_CONFIG_DIR", ""),
			AuthUpstream: getString("GATEWAY_AUTH_UPSTREAM", "http://api:8080"),
		},
	}
}

// DSN builds the PostgreSQL connection string.
func (c PostgresConfig) DSN() string {
	return "host=" + c.Host +
		" port=" + strconv.Itoa(c.Port) +
		" user=" + c.User +
		" password=" + c.Password +
		" dbname=" + c.DBName +
		" sslmode=" + c.SSLMode
}

// Addr builds the redis address host:port.
func (c RedisConfig) Addr() string {
	return c.Host + ":" + strconv.Itoa(c.Port)
}

func getString(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func getBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func splitComma(s string) []string {
	if s == "" || s == "*" {
		return []string{"*"}
	}
	result := []string{}
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			if i > start {
				result = append(result, s[start:i])
			}
			start = i + 1
		}
	}
	return result
}
