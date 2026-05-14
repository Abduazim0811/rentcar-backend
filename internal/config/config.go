package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	Env             string
	DatabaseURL     string
	RedisURL        string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	DBMaxConns      int32
	DBMinConns      int32
	MaxBodyBytes    int64
	RateLimitMax    int
	RateLimitWindow time.Duration
	TrustedProxies  []string
	MinIOEndpoint   string
	MinIOAccessKey  string
	MinIOSecretKey  string
	MinIOBucket     string
	MinIOUseSSL     bool
	ImagePublicURL  string
	SMTPHost        string
	SMTPPort        string
	SMTPUsername    string
	SMTPPassword    string
	SMTPFrom        string
	SMTPFromName    string
	SMTPUseTLS      bool
	EmailVerifyTTL  time.Duration
}

func Load() Config {
	_ = godotenv.Load()

	return Config{
		Port:            getEnv("APP_PORT", "8080"),
		Env:             getEnv("APP_ENV", "development"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/car_rental?sslmode=disable"),
		RedisURL:        getEnv("REDIS_URL", "redis://localhost:6379/0"),
		JWTSecret:       getEnv("JWT_SECRET", "rentcar"),
		AccessTokenTTL:  getDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL: getDuration("REFRESH_TOKEN_TTL", 30*24*time.Hour),
		ReadTimeout:     getDuration("HTTP_READ_TIMEOUT", 5*time.Second),
		WriteTimeout:    getDuration("HTTP_WRITE_TIMEOUT", 10*time.Second),
		IdleTimeout:     getDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout: getDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		DBMaxConns:      int32(getInt("DB_MAX_CONNS", 10)),
		DBMinConns:      int32(getInt("DB_MIN_CONNS", 1)),
		MaxBodyBytes:    int64(getInt("HTTP_MAX_BODY_BYTES", 10<<20)),
		RateLimitMax:    getInt("RATE_LIMIT_MAX_REQUESTS", 120),
		RateLimitWindow: getDuration("RATE_LIMIT_WINDOW", time.Minute),
		TrustedProxies:  getCSV("TRUSTED_PROXIES"),
		MinIOEndpoint:   getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinIOAccessKey:  getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		MinIOSecretKey:  getEnv("MINIO_SECRET_KEY", "minioadmin"),
		MinIOBucket:     getEnv("MINIO_BUCKET", "rentcar-images"),
		MinIOUseSSL:     getBool("MINIO_USE_SSL", false),
		ImagePublicURL:  getEnv("IMAGE_PUBLIC_URL", "http://localhost:8080/api/v1/uploads/images"),
		SMTPHost:        getEnv("SMTP_HOST", ""),
		SMTPPort:        getEnv("SMTP_PORT", "587"),
		SMTPUsername:    getEnv("SMTP_USERNAME", ""),
		SMTPPassword:    getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:        getEnv("SMTP_FROM", "no-reply@rentcar.local"),
		SMTPFromName:    getEnv("SMTP_FROM_NAME", "RentCar"),
		SMTPUseTLS:      getBool("SMTP_USE_TLS", false),
		EmailVerifyTTL:  getDuration("EMAIL_VERIFICATION_TTL", 10*time.Minute),
	}
}

func (c Config) Validate() error {
	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	if c.Env == "production" && (c.JWTSecret == "" || c.JWTSecret == "change-me" || c.JWTSecret == "rentcar" || len(c.JWTSecret) < 32) {
		return errors.New("JWT_SECRET must be set to a strong value with at least 32 characters in production")
	}
	if c.DBMinConns > c.DBMaxConns {
		return errors.New("DB_MIN_CONNS cannot be greater than DB_MAX_CONNS")
	}
	if c.RateLimitMax < 0 {
		return errors.New("RATE_LIMIT_MAX_REQUESTS cannot be negative")
	}
	if c.RateLimitMax > 0 && c.RateLimitWindow <= 0 {
		return errors.New("RATE_LIMIT_WINDOW must be positive when rate limiting is enabled")
	}
	if c.EmailVerifyTTL <= 0 {
		return errors.New("EMAIL_VERIFICATION_TTL must be positive")
	}

	return nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return duration
}

func getInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func getCSV(key string) []string {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}

	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}

	return result
}
