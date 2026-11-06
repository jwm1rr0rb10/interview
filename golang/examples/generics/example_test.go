package generics

import (
	"fmt"
	"strconv"
	"strings"
)

func Example_mapFilterReduce() {
	nums := []int{1, 2, 3, 4, 5, 6}
	even := Filter(nums, func(n int) bool { return n%2 == 0 })
	strs := Map(even, strconv.Itoa) // T=int, U=string выведены сами
	joined := Reduce(strs, "", func(acc, s string) string { return acc + s })
	fmt.Println(even, strs, joined)
	// Output: [2 4 6] [2 4 6] 246
}

type Rubles int64

func ExampleSum() {
	fmt.Println(Sum(1, 2, 3))
	fmt.Println(Sum(1.5, 2.5))
	fmt.Println(Sum[Rubles](100, 250)) // свой тип подходит благодаря ~int64
	// Output:
	// 6
	// 4
	// 350
}

func ExampleMaxBy() {
	words := []string{"go", "generics", "iter"}
	w, ok := MaxBy(words, func(s string) int { return len(s) })
	fmt.Println(w, ok)
	_, ok = MaxBy([]string{}, strings.ToUpper)
	fmt.Println(ok)
	// Output:
	// generics true
	// false
}

type Vec struct{ X, Y int }

func (a Vec) Add(b Vec) Vec { return Vec{a.X + b.X, a.Y + b.Y} }

func ExampleSumAdders() {
	fmt.Println(SumAdders(Vec{1, 2}, Vec{3, 4}, Vec{5, 6}))
	// Output: {9 12}
}

func ExampleSet() {
	a := NewSet(1, 2, 3, 4)
	b := NewSet(3, 4, 5)
	fmt.Println(Sorted(a.Intersect(b)))

	// Generic-метод: из Set[int] получаем Set[string].
	labels := a.Map(func(n int) string {
		if n%2 == 0 {
			return "чётное"
		}
		return "нечётное"
	})
	fmt.Println(labels.Len(), Sorted(labels))
	// Output:
	// [3 4]
	// 2 [нечётное чётное]
}

// new(выражение) из Go 1.26 заменяет хелперы вида Ptr[T](v T) *T.
func Example_newExpr() {
	type Options struct {
		Retries *int // nil означает «не задано, взять по умолчанию»
		Name    *string
	}
	o := Options{Retries: new(3), Name: new("orders")}
	fmt.Println(*o.Retries, *o.Name)
	// Output: 3 orders
}
