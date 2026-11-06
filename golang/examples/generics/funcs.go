// Пакет generics — параметры типов, ограничения и generic-методы.
package generics

import "cmp"

// Map применяет f к каждому элементу. T и U выводятся из аргументов.
func Map[T, U any](s []T, f func(T) U) []U {
	out := make([]U, 0, len(s))
	for _, v := range s {
		out = append(out, f(v))
	}
	return out
}

// Filter оставляет элементы, для которых keep возвращает true.
func Filter[T any](s []T, keep func(T) bool) []T {
	var out []T
	for _, v := range s {
		if keep(v) {
			out = append(out, v)
		}
	}
	return out
}

// Reduce сворачивает срез в одно значение.
func Reduce[T, A any](s []T, init A, f func(A, T) A) A {
	acc := init
	for _, v := range s {
		acc = f(acc, v)
	}
	return acc
}

// Number — ограничение-объединение. Тильда ~ означает «любой тип с таким базовым типом»:
// без неё свой тип `type Rubles int64` не подошёл бы.
type Number interface {
	~int | ~int32 | ~int64 | ~float32 | ~float64
}

// Sum работает с любыми числами, включая свои типы на их основе.
func Sum[T Number](nums ...T) T {
	var total T
	for _, n := range nums {
		total += n
	}
	return total
}

// MaxBy возвращает элемент с максимальным ключом. cmp.Ordered — стандартное
// ограничение для типов с операторами < и >.
func MaxBy[T any, K cmp.Ordered](s []T, key func(T) K) (T, bool) {
	var best T
	if len(s) == 0 {
		return best, false
	}
	best = s[0]
	for _, v := range s[1:] {
		if key(v) > key(best) {
			best = v
		}
	}
	return best, true
}

// Adder — ограничение, которое ссылается само на себя (разрешено с Go 1.26):
// «тип A, у которого есть метод Add(A) A».
type Adder[A Adder[A]] interface {
	Add(A) A
}

// SumAdders складывает значения любого типа с методом Add.
func SumAdders[A Adder[A]](xs ...A) A {
	var acc A
	for _, x := range xs {
		acc = acc.Add(x)
	}
	return acc
}
