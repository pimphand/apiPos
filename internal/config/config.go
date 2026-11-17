package config

import (
	"fmt"
	"os"
	"strconv"
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
