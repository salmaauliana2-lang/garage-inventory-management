package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port, GinMode, DBPath, JWTSecret string
	JWTExpiresHours                  int
	SeedDemo                         bool
	CORSOrigins                      []string
}

func Load() Config {
	hours, _ := strconv.Atoi(env("JWT_EXPIRES_HOURS", "24"))
	return Config{
		// APP_PORT untuk lokal; PORT dipakai platform deploy (mis. Hostless)
		// yang mewajibkan aplikasi listen di port yang diberikan lewat env PORT.
		Port:            env("APP_PORT", env("PORT", "8080")),
		GinMode:         env("GIN_MODE", "debug"),
		DBPath:          env("DB_PATH", "garage.db"),
		JWTSecret:       env("JWT_SECRET", "dev-only-change-me"),
		JWTExpiresHours: hours,
		SeedDemo:        env("SEED_DEMO", "true") == "true",
		CORSOrigins:     splitOrigins(env("CORS_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")),
	}
}

// splitOrigins memecah daftar origin dipisah koma, mis. "https://app.example.com,https://www.example.com".
func splitOrigins(raw string) []string {
	var out []string
	for _, o := range strings.Split(raw, ",") {
		if o = strings.TrimSpace(o); o != "" {
			out = append(out, o)
		}
	}
	return out
}

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
