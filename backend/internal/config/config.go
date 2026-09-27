package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Port          string
	DatabaseURL   string
	JWTSecret     string
	JWTTTL        time.Duration
	UploadDir     string
	CORSOrigins   []string
	AdminName     string
	AdminEmail    string
	AdminPassword string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:          env("PORT", "8080"),
		DatabaseURL:   env("DATABASE_URL", "postgres://resolveai:resolveai@localhost:5432/resolveai?sslmode=disable"),
		JWTSecret:     os.Getenv("JWT_SECRET"),
		UploadDir:     env("UPLOAD_DIR", "./uploads"),
		CORSOrigins:   strings.Split(env("CORS_ORIGINS", "http://localhost:5173"), ","),
		AdminName:     env("ADMIN_NAME", "Gestor"),
		AdminEmail:    os.Getenv("ADMIN_EMAIL"),
		AdminPassword: os.Getenv("ADMIN_PASSWORD"),
	}
	ttl, err := time.ParseDuration(env("JWT_TTL", "24h"))
	if err != nil {
		return nil, fmt.Errorf("JWT_TTL inválido: %w", err)
	}
	cfg.JWTTTL = ttl
	if len(cfg.JWTSecret) < 16 {
		return nil, fmt.Errorf("JWT_SECRET deve ter ao menos 16 caracteres")
	}
	return cfg, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
