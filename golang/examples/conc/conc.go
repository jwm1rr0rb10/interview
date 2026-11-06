// Пакет conc — горутины, каналы и select.
package conc

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Generate отдаёт числа в канал и закрывает его, когда числа кончились
// или контекст отменён. Канал только для чтения (<-chan) — так тип
// запрещает получателю писать в канал и закрывать его.
func Generate(ctx context.Context, nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out) // закрывает тот, кто пишет
		for _, n := range nums {
			select {
			case out <- n:
			case <-ctx.Done(): // без этого горутина зависнет, если читатель ушёл
				return
			}
		}
	}()
	return out
}

// Square — стадия конвейера: читает из in, пишет квадраты в out.
func Square(ctx context.Context, in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in { // цикл закончится, когда in закроют
			select {
			case out <- n * n:
			case <-ctx.Done():
				return
			}
		}
	}()
	return out
}

// Merge (fan-in) сливает несколько каналов в один.
func Merge[T any](ctx context.Context, chans ...<-chan T) <-chan T {
	out := make(chan T)
	var wg sync.WaitGroup
	for _, c := range chans {
		wg.Go(func() { // Go 1.25: Add(1) + go + Done() одной строкой
			for v := range c {
				select {
				case out <- v:
				case <-ctx.Done():
					return
				}
			}
		})
	}
	go func() {
		wg.Wait() // все писатели закончили — можно закрывать
		close(out)
	}()
	return out
}

// ErrTimeout возвращается, если ни один источник не ответил вовремя.
var ErrTimeout = errors.New("timeout")

// First запрашивает все источники параллельно и возвращает первый ответ.
// Буферизованный канал на len(sources) мест не даёт «опоздавшим» горутинам
// зависнуть на отправке, когда результат уже никому не нужен.
func First(timeout time.Duration, sources ...func() string) (string, error) {
	results := make(chan string, len(sources))
	for _, src := range sources {
		go func() { results <- src() }()
	}
	select {
	case r := <-results:
		return r, nil
	case <-time.After(timeout):
		return "", ErrTimeout
	}
}
