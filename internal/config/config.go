package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration loaded from YAML and optionally overridden
// by environment variables. Only public fields are expected to be mapped.
type Config struct {
	Server       ServerConfig       `yaml:"server"`
	Database     DatabaseConfig     `yaml:"database"`
	Logging      LoggingConfig      `yaml:"logging"`
	Telemetry    TelemetryConfig    `yaml:"telemetry"`
	Alert        AlertConfig        `yaml:"alert"`
	Retention    RetentionConfig    `yaml:"retention"`
	Notification NotificationConfig `yaml:"notification"`
}

type ServerConfig struct {
	Addr            string        `yaml:"addr"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	IdleTimeout     time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

type DatabaseConfig struct {
	Driver          string        `yaml:"driver"`
	SQLitePath      string        `yaml:"sqlite_path"`
	PostgresDSN     string        `yaml:"postgres_dsn"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

type TelemetryConfig struct {
	MaxBatchSize      int           `yaml:"max_batch_size"`
	MaxPointAge       time.Duration `yaml:"max_point_age"`
	MaxFutureSkew     time.Duration `yaml:"max_future_skew"`
	IngestRateLimit   int           `yaml:"ingest_rate_limit"`
	IngestBurst       int           `yaml:"ingest_burst"`
	QueryDefaultLimit int           `yaml:"query_default_limit"`
	QueryMaxLimit     int           `yaml:"query_max_limit"`
}

type AlertConfig struct {
	EvaluationInterval time.Duration `yaml:"evaluation_interval"`
	MaxRulesPerScan    int           `yaml:"max_rules_per_scan"`
}

type RetentionConfig struct {
	RunInterval        time.Duration `yaml:"run_interval"`
	RawRetention       time.Duration `yaml:"raw_retention"`
	AggregateRetention time.Duration `yaml:"aggregate_retention"`
	EventRetention     time.Duration `yaml:"event_retention"`
	FailureRetention   time.Duration `yaml:"failure_retention"`
	DeleteBatchSize    int           `yaml:"delete_batch_size"`
}

type NotificationConfig struct {
	MaxAttempts    int           `yaml:"max_attempts"`
	InitialBackoff time.Duration `yaml:"initial_backoff"`
	MaxBackoff     time.Duration `yaml:"max_backoff"`
	RequestTimeout time.Duration `yaml:"request_timeout"`
	WorkerCount    int           `yaml:"worker_count"`
	WebhookURL     string        `yaml:"webhook_url"`
}

// Default returns a production-sensible configuration for local development.
// The database defaults to modernc.org/sqlite so the service can run without
// any external system.
func Default() Config {
	return Config{
		Server: ServerConfig{
			Addr:            ":8080",
			ReadTimeout:     10 * time.Second,
			WriteTimeout:    30 * time.Second,
			IdleTimeout:     60 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
		Database: DatabaseConfig{
			Driver:          "sqlite",
			SQLitePath:      "./data/telemetry.db",
			PostgresDSN:     "postgres://telemetry:telemetry@localhost:5432/telemetry?sslmode=disable",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 30 * time.Minute,
		},
		Logging: LoggingConfig{Level: "info"},
		Telemetry: TelemetryConfig{
			MaxBatchSize:      500,
			MaxPointAge:       30 * 24 * time.Hour,
			MaxFutureSkew:     5 * time.Minute,
			IngestRateLimit:   500,
			IngestBurst:       500,
			QueryDefaultLimit: 200,
			QueryMaxLimit:     2000,
		},
		Alert: AlertConfig{
			EvaluationInterval: 10 * time.Second,
			MaxRulesPerScan:    500,
		},
		Retention: RetentionConfig{
			RunInterval:        5 * time.Minute,
			RawRetention:       30 * 24 * time.Hour,
			AggregateRetention: 365 * 24 * time.Hour,
			EventRetention:     90 * 24 * time.Hour,
			FailureRetention:   30 * 24 * time.Hour,
			DeleteBatchSize:    500,
		},
		Notification: NotificationConfig{
			MaxAttempts:    5,
			InitialBackoff: time.Second,
			MaxBackoff:     time.Minute,
			RequestTimeout: 5 * time.Second,
			WorkerCount:    2,
			WebhookURL:     "http://localhost:8081/webhook",
		},
	}
}

// Load reads the YAML file if path is non-empty, applies environment
// overrides, and fills missing values with defaults.
func Load(path string) (Config, error) {
	cfg := Default()
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Config{}, fmt.Errorf("parse config: %w", err)
		}
	}
	if err := applyEnv(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func applyEnv(cfg *Config) error {
	setString("SERVER_ADDR", &cfg.Server.Addr)
	setInt("SERVER_READ_TIMEOUT_SECONDS", func(v int) { cfg.Server.ReadTimeout = time.Duration(v) * time.Second })
	setInt("SERVER_WRITE_TIMEOUT_SECONDS", func(v int) { cfg.Server.WriteTimeout = time.Duration(v) * time.Second })
	setInt("SERVER_SHUTDOWN_TIMEOUT_SECONDS", func(v int) { cfg.Server.ShutdownTimeout = time.Duration(v) * time.Second })
	setString("DATABASE_DRIVER", &cfg.Database.Driver)
	setString("DATABASE_SQLITE_PATH", &cfg.Database.SQLitePath)
	setString("DATABASE_POSTGRES_DSN", &cfg.Database.PostgresDSN)
	setInt("DATABASE_MAX_OPEN_CONNS", func(v int) { cfg.Database.MaxOpenConns = v })
	setInt("DATABASE_MAX_IDLE_CONNS", func(v int) { cfg.Database.MaxIdleConns = v })
	setString("LOG_LEVEL", &cfg.Logging.Level)
	setInt("TELEMETRY_MAX_BATCH_SIZE", func(v int) { cfg.Telemetry.MaxBatchSize = v })
	setInt("TELEMETRY_INGEST_RATE_LIMIT", func(v int) { cfg.Telemetry.IngestRateLimit = v })
	setInt("TELEMETRY_INGEST_BURST", func(v int) { cfg.Telemetry.IngestBurst = v })
	setInt("TELEMETRY_QUERY_MAX_LIMIT", func(v int) { cfg.Telemetry.QueryMaxLimit = v })
	setInt("ALERT_EVALUATION_INTERVAL_SECONDS", func(v int) { cfg.Alert.EvaluationInterval = time.Duration(v) * time.Second })
	setInt("RETENTION_RUN_INTERVAL_SECONDS", func(v int) { cfg.Retention.RunInterval = time.Duration(v) * time.Second })
	setInt("RETENTION_RAW_DAYS", func(v int) { cfg.Retention.RawRetention = time.Duration(v) * 24 * time.Hour })
	setInt("RETENTION_AGGREGATE_DAYS", func(v int) { cfg.Retention.AggregateRetention = time.Duration(v) * 24 * time.Hour })
	setInt("NOTIFICATION_MAX_ATTEMPTS", func(v int) { cfg.Notification.MaxAttempts = v })
	setInt("NOTIFICATION_WORKER_COUNT", func(v int) { cfg.Notification.WorkerCount = v })
	setString("NOTIFICATION_WEBHOOK_URL", &cfg.Notification.WebhookURL)
	return nil
}

func setString(key string, dst *string) {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		*dst = v
	}
}

func setInt(key string, set func(int)) {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			set(n)
		}
	}
}
