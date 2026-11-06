package config

import (
	"log/slog"
	"strings"
	"testing"
	"time"
)

func env(m map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) { v, ok := m[k]; return v, ok }
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := load(env(map[string]string{"DATABASE_URL": "postgres://localhost/app"}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != ":8080" || cfg.LogLevel != slog.LevelInfo ||
		cfg.ShutdownTimeout != 20*time.Second || cfg.MaxWorkers != 8 {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestLoadAllErrorsAtOnce(t *testing.T) {
	_, err := load(env(map[string]string{
		"LOG_LEVEL":        "loud",
		"SHUTDOWN_TIMEOUT": "soon",
		"MAX_WORKERS":      "0",
	}))
	if err == nil {
		t.Fatal("ожидалась ошибка")
	}
	for _, want := range []string{"DATABASE_URL", "LOG_LEVEL", "SHUTDOWN_TIMEOUT", "MAX_WORKERS"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("в ошибке нет %s: %v", want, err)
		}
	}
}

// Публичная Load читает настоящее окружение; t.Setenv восстановит его после теста.
func TestLoadFromEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://db/app")
	t.Setenv("LOG_LEVEL", "debug")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogLevel != slog.LevelDebug {
		t.Fatalf("LogLevel = %v", cfg.LogLevel)
	}
}

func TestLogValueHidesSecrets(t *testing.T) {
	var b strings.Builder
	log := slog.New(slog.NewTextHandler(&b, nil))
	log.Info("config", "cfg", Config{DatabaseURL: "postgres://user:secret@db/app"})
	if strings.Contains(b.String(), "secret") {
		t.Fatalf("секрет попал в лог: %s", b.String())
	}
}
