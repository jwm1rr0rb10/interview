package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrNotFound = errors.New("task not found")

type Task struct {
	ID        int64
	Title     string
	Done      bool
	DueAt     sql.Null[time.Time] // NULL в базе; generic sql.Null — с Go 1.22
	CreatedAt time.Time
}

type TaskRepo struct {
	db *sql.DB
}

func NewTaskRepo(db *sql.DB) *TaskRepo { return &TaskRepo{db: db} }

// Create вставляет задачу. Плейсхолдеры $1, $2 — синтаксис PostgreSQL (в MySQL — ?).
// Параметры передаются отдельно от текста запроса — это защита от SQL-инъекций.
func (r *TaskRepo) Create(ctx context.Context, title string, due *time.Time) (Task, error) {
	t := Task{Title: title}
	if due != nil {
		t.DueAt = sql.Null[time.Time]{V: *due, Valid: true}
	}
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO tasks (title, due_at) VALUES ($1, $2) RETURNING id, created_at`,
		t.Title, t.DueAt,
	).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		return Task{}, fmt.Errorf("insert task: %w", err)
	}
	return t, nil
}

func (r *TaskRepo) Get(ctx context.Context, id int64) (Task, error) {
	var t Task
	err := r.db.QueryRowContext(ctx,
		`SELECT id, title, done, due_at, created_at FROM tasks WHERE id = $1`, id,
	).Scan(&t.ID, &t.Title, &t.Done, &t.DueAt, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Task{}, ErrNotFound // ошибка драйвера не утекает за пределы пакета
	}
	if err != nil {
		return Task{}, fmt.Errorf("select task %d: %w", id, err)
	}
	return t, nil
}

// ListOverdue показывает правильный цикл по строкам: defer Close и проверка rows.Err.
func (r *TaskRepo) ListOverdue(ctx context.Context, now time.Time, limit int) ([]Task, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, title, done, due_at, created_at
		   FROM tasks
		  WHERE NOT done AND due_at < $1
		  ORDER BY due_at
		  LIMIT $2`, now, limit)
	if err != nil {
		return nil, fmt.Errorf("query overdue: %w", err)
	}
	defer rows.Close() // без Close соединение не вернётся в пул

	var tasks []Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Done, &t.DueAt, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil { // ошибка, прервавшая цикл (обрыв соединения и т. п.)
		return nil, fmt.Errorf("iterate overdue: %w", err)
	}
	return tasks, nil
}

// CompleteAll закрывает задачи в одной транзакции: либо все, либо ни одной.
func (r *TaskRepo) CompleteAll(ctx context.Context, ids []int64) error {
	return WithTx(ctx, r.db, func(tx *sql.Tx) error {
		for _, id := range ids {
			res, err := tx.ExecContext(ctx, `UPDATE tasks SET done = TRUE WHERE id = $1`, id)
			if err != nil {
				return fmt.Errorf("complete %d: %w", id, err)
			}
			if n, err := res.RowsAffected(); err != nil {
				return err
			} else if n == 0 {
				return fmt.Errorf("complete %d: %w", id, ErrNotFound) // откат всей транзакции
			}
		}
		return nil
	})
}
