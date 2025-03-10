package config

import (
	"os"
	"strconv"

	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Port int `mapstructure:"port"`
	} `mapstructure:"server"`
	PoW struct {
		ServerSecret string `mapstructure:"pow_secret"`
		Difficulty   int    `mapstructure:"difficulty"`
	} `mapstructure:"pow"`
	API struct {
		URL string `mapstructure:"url"`
		Key string `mapstructure:"key"`
	} `mapstructure:"api"`
}

func LoadConfig(configPath string) (*Config, error) {
	var config Config

	viper.SetDefault("server.port", 8080)
	viper.SetDefault("pow.secret_secret", "default-secret-key")
	viper.SetDefault("pow.difficulty", 20)

	if configPath != "" {
		viper.SetConfigFile(configPath)
		if err := viper.ReadInConfig(); err != nil {
			return nil, err
		}
	}

	// Настраиваем переменные окружения
	viper.AutomaticEnv()
	viper.SetEnvPrefix("")

	viper.BindEnv("api.url", "API_URL")
	viper.BindEnv("api.key", "API_KEY")
	viper.BindEnv("pow.secret", "POW_SECRET")
	viper.BindEnv("pow.difficulty", "POW_DIFFICULTY")

	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	if apiURL := os.Getenv("API_URL"); apiURL != "" {
		config.API.URL = apiURL
	}
	if apiKey := os.Getenv("API_KEY"); apiKey != "" {
		config.API.Key = apiKey
	}
	if serverSecret := os.Getenv("POW_SECRET"); serverSecret != "" {
		config.PoW.ServerSecret = serverSecret
	}
	if powDifficulty := os.Getenv("POW_DIFFICULTY"); powDifficulty != "" {
		if difficulty, err := strconv.Atoi(powDifficulty); err == nil {
			config.PoW.Difficulty = difficulty
		}
	}

	return &config, nil
}
