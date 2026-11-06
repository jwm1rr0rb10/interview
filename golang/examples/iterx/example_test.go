package iterx

import (
	"fmt"
	"iter"
	"maps"
	"slices"
	"strings"
)

func ExampleFibonacci() {
	for n := range Fibonacci() {
		if n > 50 {
			break // yield вернёт false, и генератор завершится
		}
		fmt.Print(n, ",")
	}
	fmt.Println()
	// Output: 0,1,1,2,3,5,8,13,21,34,
}

func ExampleFilter() {
	even := Filter(Fibonacci(), func(n int) bool { return n%2 == 0 })
	fmt.Println(slices.Collect(Take(even, 5))) // Collect собирает итератор в срез
	// Output: [0 2 8 34 144]
}

func ExampleTree_All() {
	t := &Tree[int]{
		Value: 5,
		Left:  &Tree[int]{Value: 2, Right: &Tree[int]{Value: 3}},
		Right: &Tree[int]{Value: 8},
	}
	for v := range t.All() {
		fmt.Print(v, ";")
		if v == 5 {
			break // ранняя остановка рекурсивного обхода
		}
	}
	fmt.Println()
	fmt.Println(slices.Collect(t.All()))
	// Output:
	// 2;3;5;
	// [2 3 5 8]
}

func ExampleEnumerate() {
	for i, w := range Enumerate(strings.FieldsSeq("ноль один два")) {
		fmt.Println(i, w)
	}
	// Output:
	// 0 ноль
	// 1 один
	// 2 два
}

// Итераторы стандартной библиотеки: slices, maps, strings.
func Example_stdlib() {
	prices := map[string]int{"чай": 120, "кофе": 250, "сок": 180}
	fmt.Println(slices.Sorted(maps.Keys(prices)))

	for i, v := range slices.Backward([]string{"a", "b", "c"}) {
		fmt.Print(i, v, ";")
	}
	fmt.Println()

	for chunk := range slices.Chunk([]int{1, 2, 3, 4, 5}, 2) {
		fmt.Print(chunk)
	}
	fmt.Println()

	for line := range strings.Lines("первая\nвторая\n") {
		fmt.Printf("%q ", line) // строки вместе с \n
	}
	fmt.Println()
	// Output:
	// [кофе сок чай]
	// 2c;1b;0a;
	// [1 2][3 4][5]
	// "первая\n" "вторая\n"
}

// iter.Pull превращает push-итератор в pull: значения забирают по одному.
// Удобно, чтобы идти по двум последовательностям одновременно.
func Example_pull() {
	next, stop := iter.Pull(Fibonacci())
	defer stop() // обязательно: иначе ресурсы итератора не освободятся
	for range 3 {
		v, ok := next()
		fmt.Print(v, ok, " ")
	}
	fmt.Println()
	// Output: 0 true 1 true 1 true
}
