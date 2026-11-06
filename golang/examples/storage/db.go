// Пакет storage — работа с PostgreSQL через database/sql.
// Код не зависит от драйвера: драйвер подключает main через пустой импорт.
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Open открывает пул соединений и проверяет связь с базой.
// sql.DB — это пул, а не одно соединение: создаётся один раз на процесс.
func Open(ctx context.Context, driver, dsn string) (*sql.DB, error) {
	db, err := sql.Open(driver, dsn) // соединений ещё нет, только конфигурация
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(20)                  // потолок соединений с одного экземпляра сервиса
	db.SetMaxIdleConns(20)                  // держать прогретыми, а не открывать заново
	db.SetConnMaxLifetime(30 * time.Minute) // переоткрывать: балансировщики и failover
	db.SetConnMaxIdleTime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, nil
}

// WithTx выполняет fn в транзакции: Commit при успехе, Rollback при ошибке и панике.
func WithTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p) // откатили и пробросили дальше
		}
		if err != nil {
			_ = tx.Rollback() // ошибку отката не маскирует исходную
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
