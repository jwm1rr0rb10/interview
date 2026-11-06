// Пакет obs — логи, внутренние метрики рантайма и отладочный сервер.
package obs

import (
	"context"
	"io"
	"log/slog"
)

type requestIDKey struct{}

// WithRequestID кладёт ID запроса в контекст.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// ContextHandler — обёртка над любым slog.Handler: добавляет в каждую запись
// поля из контекста. Тогда вызовы log.InfoContext(ctx, ...) не повторяют request_id вручную.
type ContextHandler struct {
	slog.Handler
}

func (h ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		r.AddAttrs(slog.String("request_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

// WithAttrs и WithGroup обязаны возвращать снова обёртку, иначе после
// log.With(...) контекстные поля пропадут.
func (h ContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return ContextHandler{h.Handler.WithAttrs(attrs)}
}

func (h ContextHandler) WithGroup(name string) slog.Handler {
	return ContextHandler{h.Handler.WithGroup(name)}
}

// NewLogger: JSON для продакшена (его разбирают Loki, ELK, Cloud Logging),
// текст — для локальной разработки.
func NewLogger(w io.Writer, level slog.Leveler, json bool) *slog.Logger {
	opts := &slog.HandlerOptions{Level: level}
	var h slog.Handler = slog.NewTextHandler(w, opts)
	if json {
		h = slog.NewJSONHandler(w, opts)
	}
	return slog.New(ContextHandler{h})
}

// Secret не попадёт в лог: slog вызовет LogValue вместо печати значения.
type Secret string

func (Secret) LogValue() slog.Value { return slog.StringValue("***") }
