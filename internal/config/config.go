package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	SQS      SQSConfig      `mapstructure:"sqs"`
	AWS      AWSConfig      `mapstructure:"aws"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type DatabaseConfig struct {
	Name     string `mapstructure:"name"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
}

type RedisConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type SQSConfig struct {
	TransactionQueue    string `mapstructure:"transactionQueue"`
	TransactionDLQ      string `mapstructure:"transactionDLQ"`
	NotificationQueue   string `mapstructure:"notificationQueue"`
	NotificationDLQ     string `mapstructure:"notificationDLQ"`
	MaxNumberOfMessages int    `mapstructure:"maxNumberOfMessages"`
	WaitTimeSeconds     int    `mapstructure:"waitTimeSeconds"`
}

type AWSConfig struct {
	Host   string `mapstructure:"host"`
	Region string `mapstructure:"region"`
}

func Load() (*Config, error) {
	v := viper.New()
	var config Config

	// config file locations
	v.AddConfigPath(".")
	v.AddConfigPath("./")
	v.AddConfigPath("../../") // if running from cmd/

	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// env vars
	v.AutomaticEnv()
	v.SetEnvPrefix("FINSYS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// defaults
	v.SetDefault("server.host", "localhost")
	v.SetDefault("server.port", 8080)

	err := v.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("Error reading config file, %s", err)
	}

	err = v.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("Unable to decode into struct, %v", err)
	}

	fmt.Printf("\nconfig: %+v\n", &config)

	return &config, nil
}
