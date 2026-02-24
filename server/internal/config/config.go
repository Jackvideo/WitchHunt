package config

import (
	"os"
)

type Config struct {
	ServerPort string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	JWTSecret  string
}

func Load() *Config {
	return &Config{
		ServerPort: envOrDefault("SERVER_PORT", "8080"),
		DBHost:     envOrDefault("DB_HOST", "127.0.0.1"),
		DBPort:     envOrDefault("DB_PORT", "3306"),
		DBUser:     envOrDefault("DB_USER", "root"),
		DBPassword: envOrDefault("DB_PASSWORD", "123456"),
		DBName:     envOrDefault("DB_NAME", "witchhunt"),
		JWTSecret:  envOrDefault("JWT_SECRET", "witchhunt-dev-secret-change-me"),
	}
}

func (c *Config) DSN() string {
	return c.DBUser + ":" + c.DBPassword + "@tcp(" + c.DBHost + ":" + c.DBPort + ")/" + c.DBName + "?charset=utf8mb4&parseTime=True&loc=Local"
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
