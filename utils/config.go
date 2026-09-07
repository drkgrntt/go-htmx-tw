package utils

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment         string `mapstructure:"ENVIRONMENT"`
	ServerPort          string `mapstructure:"PORT"`
	MgApiKey            string `mapstructure:"MG_API_KEY"`
	MgWebhookSigningKey string `mapstructure:"MG_WEBHOOK_SIGNING_KEY"`
	MgDomain            string `mapstructure:"MG_DOMAIN"`
	RecipientEmail      string `mapstructure:"RECIPIENT_EMAIL"`

	DbHost     string `mapstructure:"DB_HOST"`
	DbUser     string `mapstructure:"DB_USER"`
	DbPassword string `mapstructure:"DB_PASSWORD"`
	DbName     string `mapstructure:"DB_NAME"`
	DbPort     string `mapstructure:"DB_PORT"`
}

var ConfigInstance Config

func LoadConfig(path string) (config Config, err error) {
	// godotenv.Load populates process env vars from a .env file when one is
	// present (local/air dev). In Docker the vars are already injected via
	// `env_file` in docker-compose.yml and no .env file exists on disk, so a
	// missing file here is expected, not fatal — os.Getenv below reads
	// whatever's already in the environment either way.
	if loadErr := godotenv.Load(); loadErr != nil && !os.IsNotExist(loadErr) {
		log.Fatal("Error loading .env file:", loadErr)
	}

	config = Config{
		Environment:         os.Getenv("ENVIRONMENT"),
		ServerPort:          os.Getenv("PORT"),
		MgApiKey:            os.Getenv("MG_API_KEY"),
		MgWebhookSigningKey: os.Getenv("MG_WEBHOOK_SIGNING_KEY"),
		MgDomain:            os.Getenv("MG_DOMAIN"),
		RecipientEmail:      os.Getenv("RECIPIENT_EMAIL"),

		DbHost:     os.Getenv("DB_HOST"),
		DbUser:     os.Getenv("DB_USER"),
		DbPassword: os.Getenv("DB_PASSWORD"),
		DbName:     os.Getenv("DB_NAME"),
		DbPort:     os.Getenv("DB_PORT"),
	}

	ConfigInstance = config
	return
}

func GetConfig() Config {
	return ConfigInstance
}
