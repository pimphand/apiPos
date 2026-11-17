package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"apiPos/pkg/timeutil"
)

// Config holds all configuration for the application.
type Config struct {
	App    AppConfig
	GRPC   GRPCConfig
	HTTP   HTTPConfig
	DB     DatabaseConfig
	JWT    JWTConfig
	Format FormatConfig
}

// AppConfig holds application-level configuration.
type AppConfig struct {
	Env string
}

// GRPCConfig holds gRPC server configuration.
type GRPCConfig struct {
	Port int
}

// HTTPConfig holds HTTP server configuration.
type HTTPConfig struct {
	Port int
}

// DatabaseConfig holds PostgreSQL configuration.
type DatabaseConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

// JWTConfig holds JWT configuration.
type JWTConfig struct {
	Secret     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

// FormatConfig holds the layouts used when the API renders dates and
// timestamps. Both are Go time layouts (not strftime), so adjusting the API
// output everywhere only requires changing APP_DATE_FORMAT /
// APP_DATETIME_FORMAT.
type FormatConfig struct {
	Date     string
	DateTime string
}

// DSN returns the PostgreSQL connection string.
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.SSLMode,
	)
}

// minJWTSecretBytes is the shortest JWT secret accepted at startup. HS256
// secrets shorter than the 256-bit hash output reduce the effective key space.
const minJWTSecretBytes = 32

// placeholderJWTSecrets are the values shipped by examples and compose files.
// They are publicly known, so accepting them would let anyone forge tokens.
var placeholderJWTSecrets = []string{
	"change-me",
	"changeme",
	"change_me",
	"jwt-secret",
	"your-secret-key",
}

// Validate rejects configuration that would start the server in an unsafe
// state. Call it at startup, before opening the database or network listeners,
// so a misconfigured deployment fails fast instead of issuing tokens that
// anyone can forge.
func (c *Config) Validate() error {
	secret := strings.TrimSpace(c.JWT.Secret)
	if secret == "" {
		return fmt.Errorf("JWT_SECRET is empty; set it to a unique random value (for example: openssl rand -base64 48)")
	}

	for _, placeholder := range placeholderJWTSecrets {
		if strings.EqualFold(secret, placeholder) {
			return fmt.Errorf("JWT_SECRET still uses the public placeholder %q; set it to a unique random value (for example: openssl rand -base64 48)", placeholder)
		}
	}

	if len(secret) < minJWTSecretBytes {
		return fmt.Errorf("JWT_SECRET must be at least %d bytes, got %d", minJWTSecretBytes, len(secret))
	}

	return nil
}

// Load reads configuration from environment variables.
func Load() *Config {
	return &Config{
		App: AppConfig{
			Env: getEnv("APP_ENV", "development"),
		},
		GRPC: GRPCConfig{
			Port: getEnvAsInt("GRPC_PORT", 18733),
		},
		HTTP: HTTPConfig{
			Port: getEnvAsInt("HTTP_PORT", 18734),
		},
		DB: DatabaseConfig{
			Host:     getEnv("DATABASE_HOST", "localhost"),
			Port:     getEnvAsInt("DATABASE_PORT", 5432),
			Name:     getEnv("DATABASE_NAME", "pos"),
			User:     getEnv("DATABASE_USER", "postgres"),
			Password: getEnv("DATABASE_PASSWORD", "postgres"),
			SSLMode:  getEnv("DATABASE_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			// The fallback exists only so Validate can report a precise error about
			// the placeholder instead of a generic missing value.
			Secret:     getEnv("JWT_SECRET", "change-me"),
			AccessTTL:  getEnvAsDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL: getEnvAsDuration("JWT_REFRESH_TTL", 720*time.Hour),
		},
		Format: FormatConfig{
			Date:     getEnv("APP_DATE_FORMAT", timeutil.DefaultDateLayout),
			DateTime: getEnv("APP_DATETIME_FORMAT", timeutil.DefaultDateTimeLayout),
		},
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvAsDuration(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return fallback
}
