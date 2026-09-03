package config

import (
	"bufio"
	"os"
	"strings"
)

func LoadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		_ = os.Setenv(key, val)
	}
}

type Config struct {
	Addr          string
	DatabaseURL   string
	JWTSecret     string
	MigrationsDir string
	CORSOrigins   []string
}

func Load() Config {
	return Config{
		Addr:          envOr("ADDR", ":8197"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		JWTSecret:     envOr("JWT_SECRET", "dev-jwt-secret-change-me"),
		MigrationsDir: envOr("MIGRATIONS_DIR", "migrations"),
		CORSOrigins:   splitCSV(envOr("CORS_ORIGINS", "http://localhost:5197,http://127.0.0.1:5197")),
	}
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
