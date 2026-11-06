package syncx

import (
	"sync"
	"testing"
	"time"
)

// Запускай с -race: детектор проверит, что доступ синхронизирован.
func TestCounterConcurrent(t *testing.T) {
	var c Counter
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			for range 100 {
				c.Inc("hits")
			}
		})
	}
	wg.Wait()
	if got := c.Get("hits"); got != 10_000 {
		t.Fatalf("hits = %d, want 10000", got)
	}
}

func TestStats(t *testing.T) {
	var s Stats
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() { s.Requests.Add(1) })
	}
	wg.Wait()
	s.Ready.Store(true)
	if s.Requests.Load() != 50 || !s.Ready.Load() {
		t.Fatalf("requests=%d ready=%v", s.Requests.Load(), s.Ready.Load())
	}
}

func TestCacheTTL(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	c := NewCache[string, int](time.Minute)
	c.now = func() time.Time { return now } // управляемые часы

	c.Set("a", 1)
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatalf("Get = %v, %v", v, ok)
	}
	now = now.Add(2 * time.Minute)
	if _, ok := c.Get("a"); ok {
		t.Fatal("значение должно было протухнуть")
	}
}

func TestCacheConcurrentLoad(t *testing.T) {
	c := NewCache[int, int](time.Hour)
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Go(func() {
			v, err := c.GetOrLoad(i%5, func(k int) (int, error) { return k * k, nil })
			if err != nil || v != (i%5)*(i%5) {
				t.Errorf("GetOrLoad(%d) = %d, %v", i%5, v, err)
			}
		})
	}
	wg.Wait()
}

func TestOnceValue(t *testing.T) {
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() { _ = Config()["env"] })
	}
	wg.Wait()
	if n := loadCount.Load(); n != 1 {
		t.Fatalf("загрузка выполнена %d раз, want 1", n)
	}
}
