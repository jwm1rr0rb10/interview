package generics

import (
	"cmp"
	"iter"
	"maps"
	"slices"
)

// Set — generic-тип: множество на основе map. Ключ обязан быть comparable.
type Set[T comparable] struct {
	m map[T]struct{}
}

// NewSet создаёт множество из элементов.
func NewSet[T comparable](items ...T) *Set[T] {
	s := &Set[T]{m: make(map[T]struct{}, len(items))}
	for _, v := range items {
		s.Add(v)
	}
	return s
}

func (s *Set[T]) Add(v T)          { s.m[v] = struct{}{} }
func (s *Set[T]) Has(v T) bool     { _, ok := s.m[v]; return ok }
func (s *Set[T]) Len() int         { return len(s.m) }
func (s *Set[T]) All() iter.Seq[T] { return maps.Keys(s.m) } // итератор, модуль 8

// Intersect — обычный метод: использует параметр типа T самого Set.
func (s *Set[T]) Intersect(other *Set[T]) *Set[T] {
	out := NewSet[T]()
	for v := range s.m {
		if other.Has(v) {
			out.Add(v)
		}
	}
	return out
}

// Map — generic-метод (Go 1.27): у метода свой параметр типа U.
// Раньше такое приходилось писать свободной функцией MapSet[T, U](s, f).
func (s *Set[T]) Map[U comparable](f func(T) U) *Set[U] {
	out := NewSet[U]()
	for v := range s.m {
		out.Add(f(v))
	}
	return out
}

// Sorted возвращает элементы по возрастанию. Свободная функция, потому что
// ограничение cmp.Ordered строже, чем comparable у самого Set.
func Sorted[T cmp.Ordered](s *Set[T]) []T {
	return slices.Sorted(s.All())
}
