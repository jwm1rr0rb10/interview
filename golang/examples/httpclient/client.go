// Пакет httpclient — клиент внешнего API: таймауты, ретраи, идемпотентность.
package httpclient

import (
	"bytes"
	"context"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"time"
	"uuid"
)

// StatusError — ответ с кодом не 2xx.
type StatusError struct {
	Code int
	Body string
}

func (e *StatusError) Error() string { return fmt.Sprintf("http %d: %s", e.Code, e.Body) }

type Client struct {
	BaseURL    string
	HTTP       *http.Client
	MaxRetries int
	BaseDelay  time.Duration
}

// New настраивает транспорт под один внешний сервис.
func New(baseURL string) *Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.MaxIdleConns = 100
	tr.MaxIdleConnsPerHost = 50 // по умолчанию 2: под нагрузкой соединения постоянно переоткрываются
	tr.IdleConnTimeout = 90 * time.Second
	return &Client{
		BaseURL:    baseURL,
		HTTP:       &http.Client{Transport: tr, Timeout: 10 * time.Second}, // общий таймаут попытки
		MaxRetries: 3,
		BaseDelay:  200 * time.Millisecond,
	}
}

// PostJSON отправляет in и декодирует ответ в out. Ключ идемпотентности один
// на все попытки: сервер по нему поймёт, что это повтор, и не создаст заказ дважды.
func (c *Client) PostJSON(ctx context.Context, path string, in, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	key := uuid.NewV7().String()
	return c.do(ctx, func() (*http.Request, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", key)
		return req, nil
	}, out)
}

// GetJSON — GET идемпотентен по определению, его можно повторять всегда.
func (c *Client) GetJSON(ctx context.Context, path string, out any) error {
	return c.do(ctx, func() (*http.Request, error) {
		return http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	}, out)
}

func (c *Client) do(ctx context.Context, newReq func() (*http.Request, error), out any) error {
	var lastErr error
	for attempt := 0; ; attempt++ {
		req, err := newReq() // новый запрос на каждую попытку: тело читается один раз
		if err != nil {
			return err
		}
		wait, err := c.once(req, out)
		if err == nil {
			return nil
		}
		lastErr = err
		if wait < 0 || attempt == c.MaxRetries {
			return lastErr
		}
		if wait == 0 { // сервер не подсказал паузу — экспонента с джиттером
			d := c.BaseDelay << attempt
			wait = d/2 + rand.N(d/2)
		}
		t := time.NewTimer(wait)
		select {
		case <-t.C:
		case <-ctx.Done():
			t.Stop()
			return errors.Join(lastErr, ctx.Err())
		}
	}
}

// once выполняет одну попытку. wait < 0 — не повторять; 0 — повторить с backoff;
// > 0 — повторить через указанное сервером время (Retry-After).
func (c *Client) once(req *http.Request, out any) (wait time.Duration, err error) {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		if req.Context().Err() != nil {
			return -1, err // отменили мы сами — повторять бессмысленно
		}
		if _, ok := errors.AsType[net.Error](err); ok {
			return 0, err // таймаут, обрыв соединения — временная ошибка
		}
		return -1, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if out == nil {
			return 0, nil
		}
		if err := json.UnmarshalRead(io.LimitReader(resp.Body, 10<<20), out); err != nil {
			return -1, fmt.Errorf("decode response: %w", err)
		}
		return 0, nil
	}

	msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10)) // не читаем гигабайт ошибки
	serr := &StatusError{Code: resp.StatusCode, Body: string(msg)}
	switch {
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable:
		return retryAfter(resp.Header.Get("Retry-After")), serr
	case resp.StatusCode >= 500:
		return 0, serr
	default:
		return -1, serr // 4xx — ошибка запроса, повтор не поможет
	}
}

// retryAfter понимает форму в секундах; дату (HTTP-date) для краткости не разбираем.
func retryAfter(h string) time.Duration {
	if s, err := strconv.Atoi(h); err == nil && s >= 0 && s <= 60 {
		return time.Duration(s) * time.Second
	}
	return 0
}
