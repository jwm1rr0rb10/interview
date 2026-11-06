package perf

import (
	"fmt"
	"testing"
	"unsafe"
)

var nums = func() []int {
	s := make([]int, 1000)
	for i := range s {
		s[i] = i
	}
	return s
}()

func BenchmarkJoinPlus(b *testing.B) {
	for b.Loop() {
		JoinPlus(nums)
	}
}

func BenchmarkJoinBuilder(b *testing.B) {
	for b.Loop() {
		JoinBuilder(nums)
	}
}

func BenchmarkJoinAppend(b *testing.B) {
	for b.Loop() {
		JoinAppend(nums)
	}
}

func BenchmarkSquaresNoPrealloc(b *testing.B) {
	for b.Loop() {
		SquaresNoPrealloc(10_000)
	}
}

func BenchmarkSquaresPrealloc(b *testing.B) {
	for b.Loop() {
		SquaresPrealloc(10_000)
	}
}

func BenchmarkRender(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		Render("gopher", 42)
	}
}

func TestJoinVariantsEqual(t *testing.T) {
	a, b, c := JoinPlus(nums[:20]), JoinBuilder(nums[:20]), JoinAppend(nums[:20])
	if a != b || b != c {
		t.Fatalf("реализации расходятся: %q %q %q", a, b, c)
	}
	if stackPoint() != 3 || heapPoint().Y != 2 || toAny(1) != 1 || Render("x", 1) != "hello, x #1" {
		t.Fatal("неверный результат")
	}
}

// Порядок полей влияет на размер структуры из-за выравнивания.
func Example_fieldAlignment() {
	type bad struct {
		a bool  // 1 байт + 7 байт выравнивания
		b int64 // 8
		c bool  // 1 + 7
	}
	type good struct {
		b int64 // 8
		a bool  // 1
		c bool  // 1 + 6 выравнивания в конце
	}
	fmt.Println(unsafe.Sizeof(bad{}), unsafe.Sizeof(good{}))
	// Output: 24 16
}
