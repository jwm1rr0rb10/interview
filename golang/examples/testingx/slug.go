// Пакет testingx — код, на котором показаны приёмы тестирования.
package testingx

import "strings"

var translit = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "e",
	'ж': "zh", 'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
	'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u",
	'ф': "f", 'х': "h", 'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "sch", 'ъ': "",
	'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
}

// Slugify делает из заголовка часть URL: "Привет, Go 1.27!" → "privet-go-1-27".
func Slugify(s string) string {
	var b strings.Builder
	dash := false // нужен ли дефис перед следующим символом
	for _, r := range strings.ToLower(s) {
		var part string
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			part = string(r)
		default:
			if t, ok := translit[r]; ok {
				part = t
			}
		}
		if part == "" {
			if _, isLetter := translit[r]; !isLetter {
				dash = b.Len() > 0
			}
			continue
		}
		if dash {
			b.WriteByte('-')
			dash = false
		}
		b.WriteString(part)
	}
	return b.String()
}
