// Пакет options — функциональные опции и другие идиомы конструкторов.
package options

import (
	"errors"
	"log/slog"
	"time"
)

// Client — тип с несколькими необязательными настройками.
type Client struct {
	addr    string
	timeout time.Duration
	retries int
	logger  *slog.Logger
}

// Option — функциональная опция: функция, которая настраивает Client.
type Option func(*Client) error

func WithTimeout(d time.Duration) Option {
	return func(c *Client) error {
		if d <= 0 {
			return errors.New("timeout must be positive")
		}
		c.timeout = d
		return nil
	}
}

func WithRetries(n int) Option {
	return func(c *Client) error {
		c.retries = n
		return nil
	}
}

func WithLogger(l *slog.Logger) Option {
	return func(c *Client) error {
		c.logger = l
		return nil
	}
}

// New: обязательные параметры — аргументами, необязательные — опциями.
// Новые опции добавляются без поломки существующих вызовов.
func New(addr string, opts ...Option) (*Client, error) {
	c := &Client{ // разумные значения по умолчанию
		addr:    addr,
		timeout: 5 * time.Second,
		retries: 3,
		logger:  slog.New(slog.DiscardHandler),
	}
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// Config — альтернатива опциям: структура настроек с полезными нулевыми значениями.
// Проще, когда настроек много и их удобно читать из файла.
type Config struct {
	Timeout time.Duration // 0 — значение по умолчанию
	Retries int
}

func NewFromConfig(addr string, cfg Config) *Client {
	c, _ := New(addr) // значения по умолчанию
	if cfg.Timeout > 0 {
		c.timeout = cfg.Timeout
	}
	if cfg.Retries > 0 {
		c.retries = cfg.Retries
	}
	return c
}
