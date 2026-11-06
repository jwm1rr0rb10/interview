package patterns

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"
)

type permanentError struct{ err error }

func (e permanentError) Error() string { return e.err.Error() }
func (e permanentError) Unwrap() error { return e.err }

// Permanent помечает ошибку как неповторяемую: Retry сразу её вернёт.
func Permanent(err error) error { return permanentError{err} }

// Retry повторяет fn с экспоненциальной паузой и джиттером:
// base, 2*base, 4*base... каждая пауза случайно выбирается из [d/2, d).
// Джиттер нужен, чтобы тысячи клиентов не ретраили синхронно.
func Retry(ctx context.Context, attempts int, base time.Duration, fn func(context.Context) error) error {
	var err error
	for i := range attempts {
		if err = fn(ctx); err == nil {
			return nil
		}
		if _, ok := errors.AsType[permanentError](err); ok {
			return err
		}
		if i == attempts-1 {
			break
		}
		d := base << i
		pause := d/2 + rand.N(d/2)
		t := time.NewTimer(pause)
		select {
		case <-t.C:
		case <-ctx.Done():
			t.Stop()
			return errors.Join(err, context.Cause(ctx))
		}
	}
	return err
}
