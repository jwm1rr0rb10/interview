// Пакет errs — идиоматичная обработка ошибок.
package errs

import (
	"errors"
	"fmt"
	"strings"
)

// Сигнальные (sentinel) ошибки: экспортируемые значения, с которыми сравнивают через errors.Is.
var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

// ValidationError — свой тип ошибки, когда вызывающему нужны детали.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("поле %s: %s", e.Field, e.Reason)
}

type User struct {
	ID    int
	Email string
}

var users = map[int]User{1: {ID: 1, Email: "anya@example.com"}}

// FindUser оборачивает сигнальную ошибку контекстом через %w:
// сообщение становится понятнее, а errors.Is по-прежнему находит ErrNotFound.
func FindUser(id int) (User, error) {
	u, ok := users[id]
	if !ok {
		return User{}, fmt.Errorf("find user %d: %w", id, ErrNotFound)
	}
	return u, nil
}

// Validate собирает все ошибки сразу через errors.Join.
func Validate(u User) error {
	var errs []error
	if u.ID <= 0 {
		errs = append(errs, &ValidationError{Field: "id", Reason: "должен быть положительным"})
	}
	if !strings.Contains(u.Email, "@") {
		errs = append(errs, &ValidationError{Field: "email", Reason: "нет символа @"})
	}
	return errors.Join(errs...) // nil, если ошибок нет
}

// Safely превращает панику в ошибку. Нужна на границах: обработчик HTTP-запроса,
// задача воркера, вызов плагина. Ошибки бизнес-логики паникой не передают.
func Safely(f func()) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	f()
	return nil
}
