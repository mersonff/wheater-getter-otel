package shared

import (
	"os"
	"strconv"
)

type Config struct {
	Port          string
	LogLevel      string
	LogJSON       bool
	WeatherAPIKey string
	ServiceBURL   string
	ZipkinURL     string
}

func GetConfig() Config {
	port := getEnv("PORT", "8080")
	logLevel := getEnv("LOG_LEVEL", "INFO")
	logJSON := getEnvBool("LOG_JSON", false)
	weatherAPIKey := getEnv("WEATHER_API_KEY", "")
	serviceBURL := getEnv("SERVICE_B_URL", "http://localhost:8081")
	zipkinURL := getEnv("ZIPKIN_URL", "http://localhost:9411")

	return Config{
		Port:          port,
		LogLevel:      logLevel,
		LogJSON:       logJSON,
		WeatherAPIKey: weatherAPIKey,
		ServiceBURL:   serviceBURL,
		ZipkinURL:     zipkinURL,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
