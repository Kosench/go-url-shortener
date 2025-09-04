package config

import (
	"fmt"
	"github.com/spf13/viper"
	"strings"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	App      AppConfig      `mapstructure:"app"`
	Redis    RedisConfig    `mapstructure:"redis"`
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
	BaseURL         string   `mapstructure:"base_url"`
	ShortCodeLength int      `mapstructure:"short_code_length"`
	MaxRetries      int      `mapstructure:"max_retries"`
	Environment     string   `mapstructure:"environment"`
	AllowedOrigins  []string `mapstructure:"allowed_origins"`
}

type RedisConfig struct {
	Host         string `mapstructure:"host"`
	Port         string `mapstructure:"port"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"pool_size"`
	MinIdleConns int    `mapstructure:"min_idle_conns"`
	MaxRetries   int    `mapstructure:"max_retry"`
	CacheTTL     int    `mapstructure:"cache_ttl"`
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

	// App defaults
	viper.SetDefault("app.base_url", "http://localhost:8080")
	viper.SetDefault("app.short_code_length", 6)
	viper.SetDefault("app.max_retries", 5)
	viper.SetDefault("app.environment", "development")
	viper.SetDefault("app.allowed_origins", []string{"*"})

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", "6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.pool_size", 10)
	viper.SetDefault("redis.min_idle_conns", 5)
	viper.SetDefault("redis.max_retry", 3)
	viper.SetDefault("redis.cache_ttl", 3600)

	viper.AutomaticEnv()
	viper.SetEnvPrefix("URLSHORT")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

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
		scheme := "http"
		if config.IsProduction() {
			scheme = "https"
		}
		config.App.BaseURL = fmt.Sprintf("%s://%s:%s", scheme, config.Server.Host, config.Server.Port)
	}

	return &config, nil
}

func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%s", c.Server.Host, c.Server.Port)
}

func (c *Config) GetBaseURL() string {
	return c.App.BaseURL
}

func (c *Config) IsProduction() bool {
	return strings.ToLower(c.App.Environment) == "production"
}

func (c *Config) IsDevelopment() bool {
	return strings.ToLower(c.App.Environment) == "development"
}

func (c *Config) GetAllowedOrigins() []string {
	if len(c.App.AllowedOrigins) == 0 {
		if c.IsProduction() {
			// В продакшене требуем явного указания origins
			return []string{c.App.BaseURL}
		}
		return []string{"*"}
	}
	return c.App.AllowedOrigins
}
