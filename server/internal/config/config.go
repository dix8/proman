package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv           string
	HTTPPort         string
	MySQLDSN         string
	RedisAddr        string
	RedisPassword    string
	JWTSecret        string
	JWTExpire        time.Duration
	AdminUsername    string
	AdminPassword    string
	CORSAllowOrigins []string
}

func Load() (Config, error) {
	_ = loadEnvFile(".env.local")
	_ = loadEnvFile(filepath.Join("server", ".env.local"))

	jwtExpireHours := 12
	if raw := strings.TrimSpace(os.Getenv("JWT_EXPIRE_HOURS")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return Config{}, fmt.Errorf("invalid JWT_EXPIRE_HOURS")
		}
		jwtExpireHours = parsed
	}

	cfg := Config{
		AppEnv:           getEnv("APP_ENV", "local"),
		HTTPPort:         getEnv("HTTP_PORT", "8080"),
		MySQLDSN:         strings.TrimSpace(os.Getenv("MYSQL_DSN")),
		RedisAddr:        strings.TrimSpace(os.Getenv("REDIS_ADDR")),
		RedisPassword:    strings.TrimSpace(os.Getenv("REDIS_PASSWORD")),
		JWTSecret:        strings.TrimSpace(os.Getenv("JWT_SECRET")),
		JWTExpire:        time.Duration(jwtExpireHours) * time.Hour,
		AdminUsername:    strings.TrimSpace(os.Getenv("ADMIN_USERNAME")),
		AdminPassword:    strings.TrimSpace(os.Getenv("ADMIN_PASSWORD")),
		CORSAllowOrigins: splitAndTrim(os.Getenv("CORS_ALLOW_ORIGINS")),
	}

	if cfg.MySQLDSN == "" {
		return Config{}, fmt.Errorf("MYSQL_DSN is required")
	}
	if cfg.RedisAddr == "" {
		return Config{}, fmt.Errorf("REDIS_ADDR is required")
	}
	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required")
	}
	if cfg.AdminUsername == "" {
		return Config{}, fmt.Errorf("ADMIN_USERNAME is required")
	}
	if cfg.AdminPassword == "" {
		return Config{}, fmt.Errorf("ADMIN_PASSWORD is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func splitAndTrim(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

func loadEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}

	return scanner.Err()
}
