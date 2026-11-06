package httpapi

import (
	"log/slog"
	"net/http"
	"time"
)

// NewServer задаёт таймауты. http.ListenAndServe(addr, h) их не ставит вовсе:
// медленный клиент может держать соединение бесконечно (атака Slowloris).
func NewServer(addr string, h http.Handler, log *slog.Logger) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 5 * time.Second,  // заголовки запроса
		ReadTimeout:       10 * time.Second, // весь запрос с телом
		WriteTimeout:      15 * time.Second, // от конца чтения заголовков до конца ответа
		IdleTimeout:       60 * time.Second, // keep-alive между запросами
		MaxHeaderBytes:    1 << 20,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
	}
}
