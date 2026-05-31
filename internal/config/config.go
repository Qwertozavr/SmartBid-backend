package config

import (
	"fmt"
	"os"
)

type Config struct {
	HTTPAddr    string
	DatabaseURL string
}

func Load() Config {
	serviceHost := getEnv("SERVICE_HOST", "")
	servicePort := getEnv("SERVICE_PORT", "8080")

	return Config{
		HTTPAddr:    buildHTTPAddr(serviceHost, servicePort),
		DatabaseURL: buildDatabaseURL(),
	}
}

func buildHTTPAddr(host string, port string) string {
	if host == "" {
		return ":" + port
	}
	return host + ":" + port
}

func buildDatabaseURL() string {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL != "" {
		return databaseURL
	}

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		getEnv("DB_USER", "smartbid"),
		getEnv("DB_PASSWORD", "smartbid"),
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_NAME", "smartbid"),
		getEnv("DB_SSLMODE", "disable"),
	)
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
