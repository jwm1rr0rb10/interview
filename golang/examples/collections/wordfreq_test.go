package collections

import (
	"slices"
	"testing"
)

func TestTopWords(t *testing.T) {
	text := "Go, go, GO! Срезы и map; срезы — это окна."
	got := TopWords(text, 2)
	want := []WordCount{{"go", 3}, {"срезы", 2}}
	if !slices.Equal(got, want) { // структуры сравнимы, если сравнимы все поля
		t.Errorf("TopWords() = %v, want %v", got, want)
	}
	if got := TopWords("", 5); len(got) != 0 {
		t.Errorf("пустой текст: %v", got)
	}
}
