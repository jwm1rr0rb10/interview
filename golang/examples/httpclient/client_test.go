package httpclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type order struct {
	ID     string `json:"id"`
	Amount int    `json:"amount"`
}

func newClient(url string) *Client {
	c := New(url)
	c.BaseDelay = time.Millisecond // в тестах не ждём
	return c
}

func TestRetriesServerErrorsWithSameIdempotencyKey(t *testing.T) {
	var calls atomic.Int32
	keys := make(chan string, 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keys <- r.Header.Get("Idempotency-Key")
		if calls.Add(1) < 3 {
			http.Error(w, "temporarily broken", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"o-1","amount":1990}`))
	}))
	defer srv.Close()

	var got order
	if err := newClient(srv.URL).PostJSON(t.Context(), "/orders", order{Amount: 1990}, &got); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 3 || got.ID != "o-1" {
		t.Fatalf("calls=%d got=%+v", calls.Load(), got)
	}
	close(keys)
	first := <-keys
	for k := range keys {
		if k != first || k == "" {
			t.Fatalf("ключ идемпотентности менялся между попытками: %q и %q", first, k)
		}
	}
}

func TestDoesNotRetryClientErrors(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, `{"error":"amount must be positive"}`, http.StatusUnprocessableEntity)
	}))
	defer srv.Close()

	err := newClient(srv.URL).PostJSON(t.Context(), "/orders", order{}, nil)
	serr, ok := errors.AsType[*StatusError](err)
	if !ok || serr.Code != http.StatusUnprocessableEntity || calls.Load() != 1 {
		t.Fatalf("err=%v calls=%d", err, calls.Load())
	}
}

func TestGivesUpAfterMaxRetries(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := newClient(srv.URL).GetJSON(t.Context(), "/orders/1", nil)
	if err == nil || calls.Load() != 4 { // 1 попытка + 3 повтора
		t.Fatalf("err=%v calls=%d", err, calls.Load())
	}
}

func TestRespectsContextTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(5 * time.Second):
		case <-r.Context().Done(): // клиент ушёл — сервер тоже прекращает работу
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := newClient(srv.URL).GetJSON(ctx, "/slow", nil)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > 2*time.Second {
		t.Fatalf("err=%v elapsed=%v", err, time.Since(start))
	}
}

func TestRetryAfter(t *testing.T) {
	if retryAfter("2") != 2*time.Second || retryAfter("") != 0 || retryAfter("3600") != 0 {
		t.Fatal("retryAfter")
	}
}
