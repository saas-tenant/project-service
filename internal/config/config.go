package config

import "os"

type Config struct {
	AppPort                  string
	Env                      string
	DatabaseURL              string
	InternalJWTPublicKeyPath string
	RedisAddr                string
	RedisPassword            string
	RedisDB                  int
	RabbitMQURL              string
}

func Load() *Config {
	return &Config{
		AppPort:       getEnv("APP_PORT", "4001"),
		Env:           getEnv("APP_ENV", "development"),
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		RedisAddr:     getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RabbitMQURL: getEnv(
			"RABBITMQ_URL",
			"amqp://guest:guest@127.0.0.1:5672/",
		),
		RedisDB: 0,
		InternalJWTPublicKeyPath: getEnv(
			"INTERNAL_JWT_PUBLIC_KEY_PATH",
			"./keys/jwt-public.pem",
		),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}
