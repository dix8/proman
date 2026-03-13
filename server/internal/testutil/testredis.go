package testutil

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

const defaultRedisAddr = "127.0.0.1:6379"

func OpenRedis(t *testing.T) *redis.Client {
	t.Helper()

	addr, password := resolveRedisConfig()
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       15,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("connect test redis: %v", err)
	}

	if err := client.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("flush test redis db: %v", err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_ = client.FlushDB(cleanupCtx).Err()
		_ = client.Close()
	})

	return client
}

func resolveRedisConfig() (string, string) {
	addr := strings.TrimSpace(os.Getenv("PROMAN_TEST_REDIS_ADDR"))
	if addr == "" {
		addr = strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	}
	password := strings.TrimSpace(os.Getenv("PROMAN_TEST_REDIS_PASSWORD"))
	if password == "" {
		password = strings.TrimSpace(os.Getenv("REDIS_PASSWORD"))
	}

	if addr == "" {
		addr, password = readRedisConfigFromEnvFile()
	}
	if addr == "" {
		addr = defaultRedisAddr
	}

	return addr, password
}

func readRedisConfigFromEnvFile() (string, string) {
	candidates := []string{
		filepath.Join(serverDir(), ".env.local"),
		filepath.Join(serverDir(), ".env.example"),
	}

	for _, candidate := range candidates {
		file, err := os.Open(candidate)
		if err != nil {
			continue
		}

		defer file.Close()

		var addr string
		var password string
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
			switch key {
			case "REDIS_ADDR":
				addr = value
			case "REDIS_PASSWORD":
				password = value
			}
		}

		if addr != "" {
			return addr, password
		}
	}

	return "", ""
}
