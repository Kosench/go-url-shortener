package config

import (
	"fmt"
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	App      AppConfig      `mapstructure:"app"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
}

type AppConfig struct {
	BaseURL         string `mapstructure:"base_url"`
	ShortCodeLength int    `mapstructure:"short_code_length"`
	MaxRetries      int    `mapstructure:"max_retries"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.port", "8080")

	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", "5432")
	viper.SetDefault("database.user", "urlshortener")
	viper.SetDefault("database.password", "password")
	viper.SetDefault("database.dbname", "urlshortener")

	viper.SetDefault("app.base_url", "http://localhost:8080")
	viper.SetDefault("app.short_code_length", 6)
	viper.SetDefault("app.max_retries", 5)

	// Environment variables override
	viper.AutomaticEnv()
	viper.SetEnvPrefix("URLSHORT")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	if config.App.BaseURL == "" {
		config.App.BaseURL = fmt.Sprintf("http://%s:%s", config.Server.Host, config.Server.Port)
	}

	return &config, nil
}

func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%s", c.Server.Host, c.Server.Port)
}

func (c *Config) GetBaseURL() string {
	return c.App.BaseURL
}
