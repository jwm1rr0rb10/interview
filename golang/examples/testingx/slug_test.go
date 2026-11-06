package testingx

import (
	"strings"
	"testing"
)

// Табличный тест: случаи — данные, логика проверки — одна.
func TestSlugify(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"латиница", "Hello World", "hello-world"},
		{"кириллица", "Привет, мир!", "privet-mir"},
		{"цифры и точки", "Go 1.27 вышел", "go-1-27-vyshel"},
		{"мягкий знак", "Объявление", "obyavlenie"},
		{"пробелы по краям", "  --Go--  ", "go"},
		{"пусто", "", ""},
		{"только знаки", "!!! ???", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { // подтест: go test -run 'TestSlugify/кириллица'
			t.Parallel() // подтесты выполняются параллельно
			if got := Slugify(tt.in); got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// Фаззинг: go test -fuzz=FuzzSlugify -fuzztime=30s ./testingx
// Без -fuzz функция прогоняет только начальный корпус, как обычный тест.
func FuzzSlugify(f *testing.F) {
	for _, seed := range []string{"Hello World", "Привет, мир!", "a--b", " ", "Ёж_123"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got := Slugify(s)
		// Свойства, которые должны выполняться для ЛЮБОГО входа:
		if Slugify(got) != got {
			t.Errorf("не идемпотентно: %q → %q → %q", s, got, Slugify(got))
		}
		if strings.HasPrefix(got, "-") || strings.HasSuffix(got, "-") || strings.Contains(got, "--") {
			t.Errorf("лишние дефисы: %q → %q", s, got)
		}
		for _, r := range got {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
				t.Errorf("недопустимый символ %q в %q", r, got)
			}
		}
	})
}

// Бенчмарк: go test -bench=Slugify -benchmem ./testingx
// b.Loop (Go 1.24) сам подбирает число итераций и не даёт компилятору
// выбросить «бесполезный» вызов.
func BenchmarkSlugify(b *testing.B) {
	in := "Привет, Go 1.27: generic-методы и json/v2"
	for b.Loop() {
		Slugify(in)
	}
}
