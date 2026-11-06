// Пакет collections — срезы, map, строки и структуры.
package collections

import (
	"cmp"
	"slices"
	"strings"
	"unicode"
)

// WordCount — слово и сколько раз оно встретилось.
type WordCount struct {
	Word  string
	Count int
}

// TopWords возвращает n самых частых слов текста без учёта регистра.
// При равной частоте слова идут по алфавиту — результат детерминирован.
func TopWords(text string, n int) []WordCount {
	counts := make(map[string]int)
	isSep := func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }
	for w := range strings.FieldsFuncSeq(strings.ToLower(text), isSep) {
		counts[w]++
	}

	result := make([]WordCount, 0, len(counts)) // ёмкость известна заранее
	for w, c := range counts {
		result = append(result, WordCount{Word: w, Count: c})
	}
	slices.SortFunc(result, func(a, b WordCount) int {
		return cmp.Or(
			cmp.Compare(b.Count, a.Count), // по убыванию частоты
			strings.Compare(a.Word, b.Word),
		)
	})
	if len(result) > n {
		result = result[:n]
	}
	return result
}
