package errs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

func ExampleFindUser() {
	_, err := FindUser(42)
	fmt.Println(err)
	fmt.Println(errors.Is(err, ErrNotFound)) // ищет по всей цепочке обёрток
	fmt.Println(err == ErrNotFound)          // прямое сравнение обёртку не видит
	// Output:
	// find user 42: not found
	// true
	// false
}

func ExampleValidate() {
	err := Validate(User{ID: 0, Email: "nope"})
	fmt.Println(err)

	// errors.As достаёт из цепочки первую ошибку нужного типа.
	var ve *ValidationError
	found := errors.As(err, &ve)
	fmt.Println("первое поле:", found, ve.Field)
	// errors.AsType (Go 1.26) — то же без объявления переменной.
	if ve, ok := errors.AsType[*ValidationError](err); ok {
		fmt.Println("через AsType:", ve.Field)
	}
	fmt.Println(Validate(User{ID: 1, Email: "a@b.c"}) == nil)
	// Output:
	// поле id: должен быть положительным
	// поле email: нет символа @
	// первое поле: true id
	// через AsType: id
	// true
}

// Ошибки стандартной библиотеки тоже проверяют через Is и As.
func Example_stdlibErrors() {
	_, err := os.Open("/нет/такого/файла")
	fmt.Println(errors.Is(err, fs.ErrNotExist))
	if pe, ok := errors.AsType[*fs.PathError](err); ok {
		fmt.Println(pe.Op, pe.Path)
	}
	// Output:
	// true
	// open /нет/такого/файла
}

// Несколько %w в одном Errorf (Go 1.20): ошибка оборачивает обе.
func Example_multipleWrap() {
	err := fmt.Errorf("save order: %w; rollback: %w", ErrConflict, ErrNotFound)
	fmt.Println(errors.Is(err, ErrConflict), errors.Is(err, ErrNotFound))
	// Output: true true
}

// %v вместо %w разрывает цепочку: так скрывают детали реализации от вызывающих.
func Example_opaque() {
	err := fmt.Errorf("storage failure: %v", ErrNotFound)
	fmt.Println(errors.Is(err, ErrNotFound))
	// Output: false
}

func ExampleSafely() {
	err := Safely(func() {
		var m map[string]int
		m["x"] = 1 // паника: запись в nil-map
	})
	fmt.Println(err)
	fmt.Println(Safely(func() {}))
	// Output:
	// panic: assignment to entry in nil map
	// <nil>
}
