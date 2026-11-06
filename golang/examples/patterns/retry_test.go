package patterns

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"
)

var errTemporary = errors.New("temporary")

func TestRetrySucceedsAfterFailures(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls := 0
		start := time.Now()
		err := Retry(t.Context(), 5, 100*time.Millisecond, func(context.Context) error {
			calls++
			if calls < 3 {
				return errTemporary
			}
			return nil
		})
		elapsed := time.Since(start) // виртуальное время: паузы 50–100ms и 100–200ms
		if err != nil || calls != 3 {
			t.Fatalf("err=%v calls=%d", err, calls)
		}
		if elapsed < 150*time.Millisecond || elapsed >= 300*time.Millisecond {
			t.Fatalf("elapsed = %v", elapsed)
		}
	})
}

func TestRetryStopsOnPermanent(t *testing.T) {
	calls := 0
	err := Retry(t.Context(), 5, time.Second, func(context.Context) error {
		calls++
		return Permanent(errors.New("400 bad request"))
	})
	if err == nil || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestRetryRespectsContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 250*time.Millisecond)
		defer cancel()
		err := Retry(ctx, 100, 100*time.Millisecond, func(context.Context) error {
			return errTemporary
		})
		if !errors.Is(err, errTemporary) || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("err = %v", err)
		}
	})
}
