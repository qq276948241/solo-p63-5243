package common

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	JWTSecret  string
	JWTExpire  int
	ServerPort string
}

var AppConfig *Config

func LoadConfig() error {
	_ = godotenv.Load()

	jwtExpire, _ := strconv.Atoi(getEnv("JWT_EXPIRE", "86400"))

	AppConfig = &Config{
		DBHost:     getEnv("DB_HOST", "127.0.0.1"),
		DBPort:     getEnv("DB_PORT", "3306"),
		DBUser:     getEnv("DB_USER", "root"),
		DBPassword: getEnv("DB_PASSWORD", "root"),
		DBName:     getEnv("DB_NAME", "clinic"),
		JWTSecret:  getEnv("JWT_SECRET", "clinic-secret-key"),
		JWTExpire:  jwtExpire,
		ServerPort: getEnv("SERVER_PORT", "8080"),
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
