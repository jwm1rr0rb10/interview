package obs

import (
	"bytes"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestContextHandlerAddsRequestID(t *testing.T) {
	var buf bytes.Buffer
	log := NewLogger(&buf, slog.LevelInfo, true).With("service", "orders")
	ctx := WithRequestID(t.Context(), "req-42")

	log.InfoContext(ctx, "order created", "order_id", 7, "card", Secret("4111 1111"))
	got := buf.String()
	for _, want := range []string{`"request_id":"req-42"`, `"service":"orders"`, `"order_id":7`, `"card":"***"`} {
		if !strings.Contains(got, want) {
			t.Errorf("в логе нет %s: %s", want, got)
		}
	}
	buf.Reset()
	log.Debug("не попадёт: уровень ниже Info")
	if buf.Len() != 0 {
		t.Errorf("debug-запись прошла фильтр: %s", buf.String())
	}
}

func TestDebugMux(t *testing.T) {
	srv := httptest.NewServer(DebugMux())
	defer srv.Close()
	Requests.Add(3)
	for _, path := range []string{"/debug/pprof/", "/debug/pprof/goroutine?debug=1", "/debug/pprof/goroutineleak?debug=1", "/debug/vars"} {
		resp, err := srv.Client().Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("%s: %d", path, resp.StatusCode)
		}
		if path == "/debug/vars" && !strings.Contains(string(body), `"requests_total": 3`) {
			t.Errorf("expvar: %s", body)
		}
	}
}

func TestRuntimeStats(t *testing.T) {
	s := RuntimeStats()
	if s["/sched/goroutines:goroutines"] == 0 || s["/sched/gomaxprocs:threads"] == 0 {
		t.Fatalf("stats = %v", s)
	}
}
