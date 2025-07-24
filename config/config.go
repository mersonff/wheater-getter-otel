package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port          string
	WeatherAPIKey string
	DevMode       bool
	LogJSON       bool
	LogLevel      string
}

func GetConfig() Config {
	config := Config{
		Port:          getEnv("PORT", "8080"),
		WeatherAPIKey: os.Getenv("WEATHER_API_KEY"),
		DevMode:       getEnvAsBool("DEV_MODE", false),
		LogJSON:       getEnvAsBool("LOG_JSON", false),
		LogLevel:      getEnv("LOG_LEVEL", "INFO"),
	}

	return config
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	v, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}

	return v
}
