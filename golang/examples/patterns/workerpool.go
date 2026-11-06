// Пакет patterns — context и паттерны конкурентности.
package patterns

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Process — worker pool: workers горутин обрабатывают inputs, результаты
// возвращаются в исходном порядке. Первая ошибка отменяет оставшуюся работу.
func Process[In, Out any](
	ctx context.Context,
	inputs []In,
	workers int,
	fn func(context.Context, In) (Out, error),
) ([]Out, error) {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	type job struct {
		i int
		v In
	}
	jobs := make(chan job)
	results := make([]Out, len(inputs))

	var wg sync.WaitGroup
	for range max(workers, 1) {
		wg.Go(func() {
			for j := range jobs {
				out, err := fn(ctx, j.v)
				if err != nil {
					cancel(fmt.Errorf("item %d: %w", j.i, err)) // причина отмены
					return
				}
				results[j.i] = out // у каждого задания свой индекс — гонки нет
			}
		})
	}

send:
	for i, v := range inputs {
		select {
		case jobs <- job{i, v}:
		case <-ctx.Done(): // ошибка или отмена снаружи — новые задания не раздаём
			break send
		}
	}
	close(jobs)
	wg.Wait()

	if err := context.Cause(ctx); err != nil {
		return nil, err
	}
	return results, nil
}

// RateLimited вызывает fn не чаще rps раз в секунду.
func RateLimited[T any](ctx context.Context, items []T, rps int, fn func(T)) error {
	ticker := time.NewTicker(time.Second / time.Duration(rps))
	defer ticker.Stop()
	for _, it := range items {
		select {
		case <-ticker.C:
			fn(it)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
