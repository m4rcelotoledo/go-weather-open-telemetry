package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	WeatherAPIKey string
	Port          string
}

func LoadConfig() *Config {
	// Try to load .env only in local development
	// In Cloud Run, variables will already be available in the environment
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	config := &Config{
		WeatherAPIKey: os.Getenv("WEATHER_API_KEY"),
		Port:          os.Getenv("PORT"),
	}

	// Default port if not specified
	if config.Port == "" {
		config.Port = "8080"
	}

	return config
}
