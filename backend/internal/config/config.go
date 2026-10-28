package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port, GinMode, DatabaseURL, DBPath, JWTSecret string
	JWTExpiresHours                               int
	SeedDemo                                      bool
}

func Load() Config {
	hours, _ := strconv.Atoi(env("JWT_EXPIRES_HOURS", "24"))
	return Config{
		Port:            env("APP_PORT", "8080"),
		GinMode:         env("GIN_MODE", "debug"),
		DatabaseURL:     env("DATABASE_URL", ""),
		DBPath:          env("DB_PATH", "garage.db"),
		JWTSecret:       env("JWT_SECRET", "dev-only-change-me"),
		JWTExpiresHours: hours,
		SeedDemo:        env("SEED_DEMO", "true") == "true",
	}
}

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
