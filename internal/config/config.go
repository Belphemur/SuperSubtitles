package config

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

// DefaultUserAgent is the default User-Agent string sent with all HTTP requests.
const DefaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:147.0) Gecko/20100101 Firefox/147.0"

type Config struct {
	ProxyConnectionString string `mapstructure:"proxy_connection_string"`
	SuperSubtitleDomain   string `mapstructure:"super_subtitle_domain"`
	ClientTimeout         string `mapstructure:"client_timeout"` // Go duration string like "30s", "1h", etc.
	UserAgent             string `mapstructure:"user_agent"`
	Server                struct {
		Port    int    `mapstructure:"port"`
		Address string `mapstructure:"address"`
	} `mapstructure:"server"`
	LogLevel string `mapstructure:"log_level"`
	Cache    struct {
		Size int    `mapstructure:"size"` // Maximum number of entries in the LRU cache
		TTL  string `mapstructure:"ttl"`  // Go duration string like "1h", "24h", etc.
	} `mapstructure:"cache"`
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
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

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
	if config.UserAgent == "" {
		config.UserAgent = DefaultUserAgent
	}

	return &config, nil
}

func GetConfig() *Config {
	return globalConfig
}

func GetUserAgent() string {
	if globalConfig != nil && globalConfig.UserAgent != "" {
		return globalConfig.UserAgent
	}

	return DefaultUserAgent
}

func GetLogger() zerolog.Logger {
	return logger
}
