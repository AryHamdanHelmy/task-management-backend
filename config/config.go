package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort string

	DBUser string
	DBPass string
	DBHost string
	DBPort string
	DBName string

	RedisAddr string
	RedisPass string
	RedisDB   int
}

// Load reads a .env file (if present) and environment variables,
// falling back to sensible local-dev defaults for anything missing.
func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on environment variables")
	}

	return &Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),

		DBUser: getEnv("DB_USER", "root"),
		DBPass: getEnv("DB_PASS", ""),
		DBHost: getEnv("DB_HOST", "127.0.0.1"),
		DBPort: getEnv("DB_PORT", "3306"),
		DBName: getEnv("DB_NAME", "task_management"),

		RedisAddr: getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPass: getEnv("REDIS_PASS", ""),
		RedisDB:   0,
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}