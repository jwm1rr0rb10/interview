package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"slices"
)

// Миграции встраиваются в бинарник: не нужно копировать .sql рядом с ним.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// Pending возвращает имена ещё не применённых миграций по порядку.
func Pending(fsys fs.FS, applied map[string]bool) ([]string, error) {
	names, err := fs.Glob(fsys, "migrations/*.sql")
	if err != nil {
		return nil, err
	}
	slices.Sort(names) // 001_..., 002_... — порядок задаёт префикс
	var out []string
	for _, n := range names {
		if !applied[path.Base(n)] {
			out = append(out, n)
		}
	}
	return out, nil
}

// Migrate — минимальный мигратор: каждая миграция в своей транзакции
// вместе с записью в schema_migrations. В продакшене обычно берут goose,
// golang-migrate или atlas, но принцип тот же.
func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	applied := map[string]bool{}
	rows, err := db.QueryContext(ctx, `SELECT name FROM schema_migrations`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return err
		}
		applied[n] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}

	pending, err := Pending(migrationsFS, applied)
	if err != nil {
		return err
	}
	for _, name := range pending {
		body, err := fs.ReadFile(migrationsFS, name)
		if err != nil {
			return err
		}
		err = WithTx(ctx, db, func(tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, string(body)); err != nil {
				return err
			}
			_, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, path.Base(name))
			return err
		})
		if err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
	}
	return nil
}
