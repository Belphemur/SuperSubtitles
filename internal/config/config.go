package config

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

type Config struct {
	ProxyConnectionString string `mapstructure:"proxy_connection_string"`
	SuperSubtitleDomain   string `mapstructure:"super_subtitle_domain"`
	ClientTimeout         string `mapstructure:"client_timeout"` // Go duration string like "30s", "1h", etc.
	Server                struct {
		Port    int    `mapstructure:"port"`
		Address string `mapstructure:"address"`
	} `mapstructure:"server"`
	LogLevel string `mapstructure:"log_level"`
}

var (
	globalConfig *Config
	logger       zerolog.Logger
)

func init() {
	// Initialize zerolog with console writer for human-readable output
	logger = zerolog.New(zerolog.ConsoleWriter{
		Out:     os.Stdout,
		NoColor: false,
	}).With().Timestamp().Logger()

	config, err := LoadConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load config")
	}

	// Parse and set log level from config
	level := zerolog.InfoLevel // default
	if config.LogLevel != "" {
		if parsedLevel, err := zerolog.ParseLevel(config.LogLevel); err == nil {
			level = parsedLevel
		} else {
			logger.Warn().Str("invalid_level", config.LogLevel).Msg("Invalid log level, using default 'info'")
		}
	}

	// Set the global log level
	zerolog.SetGlobalLevel(level)

	// Update logger with the configured level
	logger = logger.Level(level)

	logger.Info().Str("level", level.String()).Msg("Logging configured")
	globalConfig = config
	logger.Info().Msg("Configuration loaded successfully")
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Environment variable support
	viper.AutomaticEnv()
	viper.SetEnvPrefix("APP")

	// Add specific environment variable for log level
	_ = viper.BindEnv("log_level", "LOG_LEVEL")

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

func GetConfig() *Config {
	return globalConfig
}

func GetLogger() zerolog.Logger {
	return logger
}
