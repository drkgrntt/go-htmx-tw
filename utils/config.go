package utils

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Environment    string `mapstructure:"ENVIRONMENT"`
	ServerPort     string `mapstructure:"PORT"`
	MgApiKey       string `mapstructure:"MG_API_KEY"`
	MgDomain       string `mapstructure:"MG_DOMAIN"`
	RecipientEmail string `mapstructure:"RECIPIENT_EMAIL"`
}

var ConfigInstance Config

func LoadConfig(path string) (config Config, err error) {
	err = godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	if err != nil {
		return
	}

	config = Config{
		Environment:    os.Getenv("ENVIRONMENT"),
		ServerPort:     os.Getenv("PORT"),
		MgApiKey:       os.Getenv("MG_API_KEY"),
		MgDomain:       os.Getenv("MG_DOMAIN"),
		RecipientEmail: os.Getenv("RECIPIENT_EMAIL"),
	}

	ConfigInstance = config
	return
}

func GetConfig() Config {
	return ConfigInstance
}
