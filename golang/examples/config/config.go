// Пакет config загружает конфигурацию сервиса из переменных окружения.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr            string
	DatabaseURL     string
	LogLevel        slog.Level
	ShutdownTimeout time.Duration
	MaxWorkers      int
}

// Load читает окружение, подставляет значения по умолчанию и валидирует всё сразу:
// сервис с неверной конфигурацией должен упасть при старте, а не через час.
func Load() (Config, error) {
	return load(os.LookupEnv)
}

// load принимает функцию поиска — так тесты не зависят от реального окружения.
func load(lookup func(string) (string, bool)) (Config, error) {
	var errs []error
	get := func(key, def string) string {
		if v, ok := lookup(key); ok && v != "" {
			return v
		}
		return def
	}

	cfg := Config{
		Addr:        get("ADDR", ":8080"),
		DatabaseURL: get("DATABASE_URL", ""),
	}
	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if err := cfg.LogLevel.UnmarshalText([]byte(get("LOG_LEVEL", "info"))); err != nil {
		errs = append(errs, fmt.Errorf("LOG_LEVEL: %w", err))
	}
	d, err := time.ParseDuration(get("SHUTDOWN_TIMEOUT", "20s"))
	if err != nil || d <= 0 {
		errs = append(errs, fmt.Errorf("SHUTDOWN_TIMEOUT: invalid duration %q", get("SHUTDOWN_TIMEOUT", "")))
	}
	cfg.ShutdownTimeout = d
	n, err := strconv.Atoi(get("MAX_WORKERS", "8"))
	if err != nil || n < 1 || n > 1000 {
		errs = append(errs, fmt.Errorf("MAX_WORKERS: must be 1..1000"))
	}
	cfg.MaxWorkers = n

	if err := errors.Join(errs...); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}
	return cfg, nil
}

// LogValue скрывает секреты, если конфигурацию целиком пишут в лог.
func (c Config) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("addr", c.Addr),
		slog.String("database_url", "***"),
		slog.String("log_level", c.LogLevel.String()),
		slog.Duration("shutdown_timeout", c.ShutdownTimeout),
		slog.Int("max_workers", c.MaxWorkers),
	)
}
