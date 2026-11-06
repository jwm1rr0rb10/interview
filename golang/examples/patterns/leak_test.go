package patterns

import (
	"bytes"
	"runtime/pprof"
	"strings"
	"testing"
	"time"
)

// leakyFirst — типичная утечка: результат ждут только от первой горутины,
// остальные навсегда блокируются на отправке в небуферизованный канал.
func leakyFirst(sources []func() int) int {
	ch := make(chan int) // исправление: make(chan int, len(sources))
	for _, src := range sources {
		go func() { ch <- src() }()
	}
	return <-ch
}

// Профиль goroutineleak (Go 1.27) находит горутины, заблокированные навсегда:
// канал, на котором они ждут, недостижим ни из одной живой горутины.
func TestGoroutineLeakProfile(t *testing.T) {
	src := func() int { return 1 }
	leakyFirst([]func() int{src, src, src})
	time.Sleep(50 * time.Millisecond) // дать горутинам заблокироваться

	var buf bytes.Buffer
	if err := pprof.Lookup("goroutineleak").WriteTo(&buf, 1); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "leakyFirst") {
		t.Fatalf("утечка не найдена в профиле:\n%s", buf.String())
	}
	t.Logf("профиль:\n%s", buf.String())
}
