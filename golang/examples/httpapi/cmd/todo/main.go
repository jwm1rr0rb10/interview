// Сервер задач с graceful shutdown: go run ./httpapi/cmd/todo
package main

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gocourse/httpapi"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("server stopped with error", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	// ctx отменится по Ctrl+C или SIGTERM (так Kubernetes останавливает под).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	addr := cmp.Or(os.Getenv("ADDR"), ":8080")
	srv := httpapi.NewServer(addr, httpapi.New(httpapi.NewStore(), log), log)

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", addr)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err // например, порт занят
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	// Shutdown закрывает слушатель и ждёт завершения активных запросов.
	// Таймаут должен быть меньше terminationGracePeriodSeconds пода (по умолчанию 30 с).
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	return <-errCh
}
