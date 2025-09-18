package main

import (
	"github.com/spf13/viper"
)

type Config struct {
	ProxyConnectionString string `mapstructure:"proxy_connection_string"`
	SuperSubtitleDomain   string `mapstructure:"super_subtitle_domain"`
	Server                struct {
		Port    int    `mapstructure:"port"`
		Address string `mapstructure:"address"`
	} `mapstructure:"server"`
	Debug bool `mapstructure:"debug"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Environment variable support
	viper.AutomaticEnv()
	viper.SetEnvPrefix("APP")

	// Set defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.address", "localhost")
	viper.SetDefault("proxy_connection_string", "")
	viper.SetDefault("super_subtitle_domain", "")
	viper.SetDefault("debug", false)

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func main() {
	// Do nothing
}
