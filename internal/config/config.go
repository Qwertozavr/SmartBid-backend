package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	HTTPAddr            string
	DatabaseURL         string
	KafkaBrokers        []string
	KafkaAdCreatedTopic string
	KafkaDLQTopic       string
}

func Load() Config {
	serviceHost := getEnv("SERVICE_HOST", "")
	servicePort := getEnv("SERVICE_PORT", "8080")

	return Config{
		HTTPAddr:            buildHTTPAddr(serviceHost, servicePort),
		DatabaseURL:         buildDatabaseURL(),
		KafkaBrokers:        splitCSV(getEnv("KAFKA_BROKERS", "localhost:9092")),
		KafkaAdCreatedTopic: getEnv("KAFKA_AD_CREATED_TOPIC", "ad-created"),
		KafkaDLQTopic:       getEnv("KAFKA_DLQ_TOPIC", "ad-created-dlq"),
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

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
