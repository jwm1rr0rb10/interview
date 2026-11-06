package obs

import (
	"expvar"
	"net/http"
	"net/http/pprof"
	"runtime/metrics"
)

// DebugMux — отдельный обработчик для внутреннего порта (например, :6060).
// Пакет net/http/pprof при импорте регистрирует себя в http.DefaultServeMux,
// поэтому основной сервер никогда не должен обслуживать DefaultServeMux.
func DebugMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index) // heap, goroutine, goroutineleak, allocs...
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile) // CPU, ?seconds=30
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.Handle("/debug/vars", expvar.Handler()) // счётчики expvar в JSON
	return mux
}

// RuntimeStats читает метрики рантайма через стабильный API runtime/metrics.
func RuntimeStats() map[string]uint64 {
	samples := []metrics.Sample{
		{Name: "/sched/goroutines:goroutines"},
		{Name: "/sched/gomaxprocs:threads"},
		{Name: "/memory/classes/heap/objects:bytes"},
		{Name: "/gc/cycles/total:gc-cycles"},
	}
	metrics.Read(samples)
	out := make(map[string]uint64, len(samples))
	for _, s := range samples {
		if s.Value.Kind() == metrics.KindUint64 {
			out[s.Name] = s.Value.Uint64()
		}
	}
	return out
}

// Requests — пример счётчика expvar: виден в /debug/vars.
var Requests = expvar.NewInt("requests_total")
