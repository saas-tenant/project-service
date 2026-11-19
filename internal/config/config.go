
package config

import "os"

type Config struct {
	AppPort string
	Env     string
}

func Load() *Config {
	return &Config{
		AppPort: getEnv("APP_PORT", "4001"),
		Env:     getEnv("APP_ENV", "development"),
	}
}

// getEnv returns the environment variable if set.
// Otherwise, it returns the provided fallback.
func getEnv(key, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}
