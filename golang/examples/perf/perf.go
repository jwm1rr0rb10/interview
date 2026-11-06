// Пакет perf — примеры для профилирования, бенчмарков и escape analysis.
package perf

import (
	"bytes"
	"strconv"
	"strings"
	"sync"
)

// JoinPlus склеивает конкатенацией: каждая итерация — новая строка и копирование.
func JoinPlus(nums []int) string {
	s := ""
	for _, n := range nums {
		s = s + strconv.Itoa(n) + ","
	}
	return s
}

// JoinBuilder пишет в один растущий буфер.
func JoinBuilder(nums []int) string {
	var b strings.Builder
	b.Grow(len(nums) * 4) // предварительная ёмкость убирает перевыделения
	for _, n := range nums {
		b.WriteString(strconv.Itoa(n))
		b.WriteByte(',')
	}
	return b.String()
}

// JoinAppend избегает даже промежуточных строк от Itoa.
func JoinAppend(nums []int) string {
	buf := make([]byte, 0, len(nums)*4)
	for _, n := range nums {
		buf = strconv.AppendInt(buf, int64(n), 10)
		buf = append(buf, ',')
	}
	return string(buf)
}

// SquaresNoPrealloc растит срез через append с нуля.
func SquaresNoPrealloc(n int) []int {
	var out []int
	for i := range n {
		out = append(out, i*i)
	}
	return out
}

// SquaresPrealloc выделяет память один раз.
func SquaresPrealloc(n int) []int {
	out := make([]int, 0, n)
	for i := range n {
		out = append(out, i*i)
	}
	return out
}

var bufPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

// Render использует буфер из пула: меньше мусора под нагрузкой.
func Render(name string, n int) string {
	buf := bufPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		if buf.Cap() <= 64<<10 { // огромные буферы в пул не возвращаем
			bufPool.Put(buf)
		}
	}()
	buf.WriteString("hello, ")
	buf.WriteString(name)
	buf.WriteString(" #")
	buf.WriteString(strconv.Itoa(n))
	return buf.String()
}

type point struct{ X, Y int }

// stackPoint: значение не покидает функцию — живёт на стеке.
func stackPoint() int {
	p := point{1, 2}
	return p.X + p.Y
}

// heapPoint: указатель возвращается наружу — значение уезжает в кучу.
func heapPoint() *point {
	p := point{1, 2}
	return &p
}

// toAny: преобразование в интерфейс обычно требует аллокации.
func toAny(n int) any {
	return n
}
