package patterns

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

func TestProcessKeepsOrder(t *testing.T) {
	double := func(_ context.Context, n int) (int, error) { return n * 2, nil }
	got, err := Process(t.Context(), []int{1, 2, 3, 4, 5}, 3, double)
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range got {
		if v != (i+1)*2 {
			t.Fatalf("got %v", got)
		}
	}
}

func TestProcessFirstErrorCancelsRest(t *testing.T) {
	var calls atomic.Int32
	boom := errors.New("boom")
	fn := func(ctx context.Context, n int) (int, error) {
		calls.Add(1)
		if n == 3 {
			return 0, boom
		}
		select {
		case <-time.After(50 * time.Millisecond):
		case <-ctx.Done(): // хорошая функция сама уважает отмену
		}
		return n, nil
	}
	inputs := make([]int, 100)
	for i := range inputs {
		inputs[i] = i
	}
	_, err := Process(t.Context(), inputs, 4, fn)
	if !errors.Is(err, boom) || !strings.Contains(err.Error(), "item 3") {
		t.Fatalf("err = %v", err)
	}
	if n := calls.Load(); n >= 100 {
		t.Fatalf("ошибка не остановила обработку: %d вызовов", n)
	}
}

// synctest: внутри «пузыря» время виртуальное и идёт мгновенно,
// когда все горутины пузыря заблокированы.
func TestRateLimited(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		var n int
		err := RateLimited(t.Context(), []int{1, 2, 3, 4, 5}, 10, func(int) { n++ })
		if err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(start); n != 5 || elapsed != 500*time.Millisecond {
			t.Fatalf("n=%d elapsed=%v, want 5 и 500ms", n, elapsed)
		}
	})
}
