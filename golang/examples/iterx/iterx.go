// Пакет iterx — свои итераторы на основе iter.Seq (range по функции, Go 1.23).
package iterx

import "iter"

// Fibonacci — бесконечный итератор. Он остановится, когда цикл
// вызывающего сделает break: тогда yield вернёт false.
func Fibonacci() iter.Seq[int] {
	return func(yield func(int) bool) {
		a, b := 0, 1
		for {
			if !yield(a) {
				return
			}
			a, b = b, a+b
		}
	}
}

// Filter — адаптер: принимает итератор и возвращает итератор.
func Filter[T any](seq iter.Seq[T], keep func(T) bool) iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range seq {
			if keep(v) && !yield(v) {
				return
			}
		}
	}
}

// Take ограничивает итератор первыми n элементами.
func Take[T any](seq iter.Seq[T], n int) iter.Seq[T] {
	return func(yield func(T) bool) {
		if n <= 0 {
			return
		}
		i := 0
		for v := range seq {
			if !yield(v) {
				return
			}
			i++
			if i == n {
				return
			}
		}
	}
}

// Tree — двоичное дерево поиска.
type Tree[T any] struct {
	Left, Right *Tree[T]
	Value       T
}

// All обходит дерево по порядку. Рекурсивный обход с ранней остановкой —
// то, что раньше требовало каналов или колбэков с флагами.
func (t *Tree[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		t.walk(yield)
	}
}

func (t *Tree[T]) walk(yield func(T) bool) bool {
	if t == nil {
		return true
	}
	return t.Left.walk(yield) && yield(t.Value) && t.Right.walk(yield)
}

// Enumerate — пример iter.Seq2: пары (индекс, значение).
func Enumerate[T any](seq iter.Seq[T]) iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		i := 0
		for v := range seq {
			if !yield(i, v) {
				return
			}
			i++
		}
	}
}
