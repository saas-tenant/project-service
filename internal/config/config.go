
package config

import "os"

type Config struct {
	AppPort    string
	Env        string
	DatabaseURL string
}

func Load() *Config {
	return &Config{
		AppPort:     getEnv("APP_PORT", "4001"),
		Env:         getEnv("APP_ENV", "development"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}
