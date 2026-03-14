package config

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/Belphemur/SuperSubtitles/v2/internal/sentryio"
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
	LogLevel  string `mapstructure:"log_level"`
	LogFormat string `mapstructure:"log_format"` // Log output format: "console" (default) or "json"
	Cache     struct {
		Type  string `mapstructure:"type"` // Cache backend: "memory" (default) or "redis"
		Size  int    `mapstructure:"size"` // Maximum number of entries in the LRU cache
		TTL   string `mapstructure:"ttl"`  // Go duration string like "1h", "24h", etc.
		Redis struct {
			Address  string `mapstructure:"address"`  // Redis/Valkey server address (e.g., "localhost:6379")
			Password string `mapstructure:"password"` // Redis/Valkey password (optional)
			DB       int    `mapstructure:"db"`       // Redis/Valkey database number (default 0)
		} `mapstructure:"redis"`
	} `mapstructure:"cache"`
	Metrics struct {
		Enabled bool `mapstructure:"enabled"` // Whether to expose Prometheus metrics
		Port    int  `mapstructure:"port"`    // Port for the metrics HTTP server
	} `mapstructure:"metrics"`
	Sentry struct {
		DSN          string `mapstructure:"dsn"`           // Sentry DSN; empty disables Sentry reporting
		Environment  string `mapstructure:"environment"`   // Optional Sentry environment override
		Debug        bool   `mapstructure:"debug"`         // Enable sentry-go debug logging
		FlushTimeout string `mapstructure:"flush_timeout"` // Flush timeout during shutdown, e.g. "2s"
	} `mapstructure:"sentry"`
	Retry struct {
		MaxAttempts  int    `mapstructure:"max_attempts"`  // Total attempts including the initial try (0 uses default of 3)
		InitialDelay string `mapstructure:"initial_delay"` // Delay before the first retry, e.g. "500ms", "1s" (empty = no delay)
		MaxDelay     string `mapstructure:"max_delay"`     // Maximum retry delay with exponential back-off, e.g. "10s" (empty = use initial_delay as cap)
	} `mapstructure:"retry"`
}

var (
	globalConfig *Config
	logger       zerolog.Logger
)

func init() {
	// Initialize zerolog with console writer for human-readable output (default before config loads)
	logger = zerolog.New(zerolog.ConsoleWriter{
		Out:     os.Stdout,
		NoColor: false,
	}).With().Timestamp().Logger()

	config, err := LoadConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load config")
	}

	// Determine the base output writer from the log_format setting before
	// initializing Sentry so that Sentry-init logs already use the correct format.
	var baseWriter io.Writer
	switch config.LogFormat {
	case "json":
		baseWriter = os.Stdout
	case "console", "":
		baseWriter = zerolog.ConsoleWriter{Out: os.Stdout, NoColor: false}
	default:
		logger.Warn().Str("invalid_format", config.LogFormat).Msg("Invalid log format, using default 'console'")
		baseWriter = zerolog.ConsoleWriter{Out: os.Stdout, NoColor: false}
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

	// Rebuild the logger with the configured format and level so that
	// Sentry-init messages (below) are already using the right writer.
	logger = zerolog.New(baseWriter).With().Timestamp().Logger().Level(level)

	// Initialize Sentry after the logger is configured so any warnings or
	// info messages emitted during init use the correct log format/level.
	if err := initSentry(config); err != nil {
		logger.Warn().Err(err).Msg("Failed to initialize Sentry, continuing without it")
	}

	// When Sentry is enabled, wrap the writer so log events are automatically
	// recorded as Sentry breadcrumbs and structured logs.
	if sentryio.Enabled() {
		writer := zerolog.MultiLevelWriter(baseWriter, sentryio.NewWriter())
		logger = zerolog.New(writer).With().Timestamp().Logger().Level(level)
	}

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
	// Add specific environment variable for log format
	_ = viper.BindEnv("log_format", "LOG_FORMAT")

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

// FlushSentry flushes any queued Sentry events before shutdown.
func FlushSentry() bool {
	return sentryio.Flush()
}

func initSentry(cfg *Config) error {
	flushTimeout := 2 * time.Second
	if cfg.Sentry.FlushTimeout != "" {
		parsedTimeout, err := time.ParseDuration(cfg.Sentry.FlushTimeout)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("sentry.flush_timeout", cfg.Sentry.FlushTimeout).
				Dur("fallback", flushTimeout).
				Msg("Invalid sentry.flush_timeout value, falling back to default")
		} else {
			flushTimeout = parsedTimeout
		}
	}

	reporter, err := sentryio.New(sentryio.Config{
		DSN:          cfg.Sentry.DSN,
		Environment:  cfg.Sentry.Environment,
		Debug:        cfg.Sentry.Debug,
		FlushTimeout: flushTimeout,
	})
	if err != nil {
		return err
	}

	sentryio.SetGlobal(reporter)

	if reporter.Enabled() {
		logger.Info().
			Str("environment", cfg.Sentry.Environment).
			Str("flush_timeout", flushTimeout.String()).
			Msg("Sentry reporting enabled")
	}

	return nil
}
