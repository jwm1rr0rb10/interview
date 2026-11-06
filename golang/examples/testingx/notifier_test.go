package testingx

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeSender — ручной фейк вместо мок-фреймворка: записывает вызовы.
type fakeSender struct {
	sent []string
	err  error
}

func (f *fakeSender) Send(_ context.Context, to, msg string) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, to+": "+msg)
	return nil
}

func TestNotifyOverdue(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	tasks := []Task{
		{Title: "отчёт", Owner: "аня", Due: now.Add(-48 * time.Hour)},
		{Title: "релиз", Owner: "боря", Due: now.Add(24 * time.Hour)},
	}
	fake := &fakeSender{}
	n := &Notifier{Sender: fake, Now: func() time.Time { return now }}

	got, err := n.NotifyOverdue(t.Context(), tasks)
	if err != nil {
		t.Fatal(err)
	}
	if got != 1 || len(fake.sent) != 1 {
		t.Fatalf("sent = %d, calls = %v", got, fake.sent)
	}
	if want := `аня: задача "отчёт" просрочена с 2026-11-04`; fake.sent[0] != want {
		t.Errorf("сообщение = %q, want %q", fake.sent[0], want)
	}
}

func TestNotifyOverdueSenderError(t *testing.T) {
	smtpDown := errors.New("smtp down")
	n := &Notifier{
		Sender: &fakeSender{err: smtpDown},
		Now:    time.Now,
	}
	_, err := n.NotifyOverdue(t.Context(), []Task{{Owner: "аня", Due: time.Now().Add(-time.Hour)}})
	if !errors.Is(err, smtpDown) {
		t.Fatalf("err = %v, want обёртку smtpDown", err)
	}
}
