package testingx

import (
	"context"
	"fmt"
	"time"
)

// Sender — интерфейс, объявленный потребителем: Notifier нужен только Send.
type Sender interface {
	Send(ctx context.Context, to, msg string) error
}

type Task struct {
	Title string
	Owner string
	Due   time.Time
}

type Notifier struct {
	Sender Sender
	Now    func() time.Time // часы внедряются так же, как остальные зависимости
}

// NotifyOverdue шлёт напоминания по просроченным задачам и возвращает их число.
func (n *Notifier) NotifyOverdue(ctx context.Context, tasks []Task) (int, error) {
	sent := 0
	for _, t := range tasks {
		if !t.Due.Before(n.Now()) {
			continue
		}
		msg := fmt.Sprintf("задача %q просрочена с %s", t.Title, t.Due.Format(time.DateOnly))
		if err := n.Sender.Send(ctx, t.Owner, msg); err != nil {
			return sent, fmt.Errorf("notify %s: %w", t.Owner, err)
		}
		sent++
	}
	return sent, nil
}
