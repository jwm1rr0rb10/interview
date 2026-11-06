// Пакет syncx — мьютексы, атомики, Once и потокобезопасный кэш.
package syncx

import (
	"sync"
	"sync/atomic"
	"time"
)

// Counter защищает map мьютексом. Нулевое значение готово к работе,
// но копировать Counter нельзя: копия мьютекса — отдельный мьютекс.
type Counter struct {
	mu sync.Mutex
	m  map[string]int
}

func (c *Counter) Inc(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil {
		c.m = make(map[string]int)
	}
	c.m[key]++
}

func (c *Counter) Get(key string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.m[key]
}

// Stats — счётчики на атомиках: без блокировок, для простых чисел.
type Stats struct {
	Requests atomic.Int64
	Errors   atomic.Int64
	Ready    atomic.Bool
}

// Cache — кэш с TTL на RWMutex: много читателей одновременно,
// писатель — эксклюзивно.
type Cache[K comparable, V any] struct {
	mu    sync.RWMutex
	items map[K]entry[V]
	ttl   time.Duration
	now   func() time.Time // подменяется в тестах
}

type entry[V any] struct {
	val     V
	expires time.Time
}

func NewCache[K comparable, V any](ttl time.Duration) *Cache[K, V] {
	return &Cache[K, V]{items: make(map[K]entry[V]), ttl: ttl, now: time.Now}
}

func (c *Cache[K, V]) Get(k K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.items[k]
	if !ok || c.now().After(e.expires) {
		var zero V
		return zero, false
	}
	return e.val, true
}

func (c *Cache[K, V]) Set(k K, v V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[k] = entry[V]{val: v, expires: c.now().Add(c.ttl)}
}

// GetOrLoad не вызывает load под блокировкой: медленная загрузка под мьютексом
// остановила бы всех читателей. Цена — возможна двойная загрузка при гонке;
// если это недопустимо, используют singleflight (golang.org/x/sync/singleflight).
func (c *Cache[K, V]) GetOrLoad(k K, load func(K) (V, error)) (V, error) {
	if v, ok := c.Get(k); ok {
		return v, nil
	}
	v, err := load(k)
	if err != nil {
		return v, err
	}
	c.Set(k, v)
	return v, nil
}

// loadConfig вызывается один раз, сколько бы горутин ни обратились к Config.
var loadCount atomic.Int32

var Config = sync.OnceValue(func() map[string]string {
	loadCount.Add(1)
	return map[string]string{"env": "prod"} // например, чтение файла
})
