package basics

import (
	"errors"
	"fmt"
)

func ExampleDivMod() {
	q, r, err := DivMod(17, 5)
	fmt.Println(q, r, err)

	_, _, err = DivMod(1, 0)
	fmt.Println(errors.Is(err, ErrDivByZero))
	// Output:
	// 3 2 <nil>
	// true
}

func ExampleSum() {
	fmt.Println(Sum())
	fmt.Println(Sum(1, 2, 3))
	nums := []int{10, 20}
	fmt.Println(Sum(nums...)) // развернуть срез в аргументы
	// Output:
	// 0
	// 6
	// 30
}

func ExampleNewCounter() {
	next := NewCounter()
	other := NewCounter() // у каждого замыкания своя n
	fmt.Println(next(), next(), next(), other())
	// Output: 1 2 3 1
}

func ExampleWeekday() {
	d := Saturday
	fmt.Println(d, int(d), d.IsWeekend())
	fmt.Printf("%v %d %s\n", Monday, Monday, Weekday(42))
	// Output:
	// сб 6 true
	// пн 1 Weekday(42)
}

func ExampleGrade() {
	for _, s := range []int{95, 80, 61, 10} {
		fmt.Println(s, Grade(s))
	}
	// Output:
	// 95 отлично
	// 80 хорошо
	// 61 удовлетворительно
	// 10 неудовлетворительно
}

// Порядок выполнения defer: последним отложен — первым выполнен.
// Аргументы defer вычисляются сразу, в момент объявления.
func Example_defer() {
	for i := range 3 { // range по числу: i = 0, 1, 2
		defer fmt.Println("defer", i)
	}
	fmt.Println("тело функции")
	// Output:
	// тело функции
	// defer 2
	// defer 1
	// defer 0
}

// Именованный результат можно изменить в defer — так оборачивают ошибки.
func Example_deferNamedResult() {
	f := func() (err error) {
		defer func() {
			if err != nil {
				err = fmt.Errorf("load config: %w", err)
			}
		}()
		return errors.New("файл не найден")
	}
	fmt.Println(f())
	// Output: load config: файл не найден
}

// Затенение (shadowing): := во внутреннем блоке создаёт новую переменную.
func Example_shadowing() {
	x := 1
	if true {
		x := 2 // новая x, внешняя не меняется
		x++
		_ = x
	}
	fmt.Println(x)
	// Output: 1
}

// С Go 1.22 у каждой итерации цикла своя переменная.
func Example_loopVar() {
	var funcs []func() int
	for i := 1; i <= 3; i++ { // классическая ловушка до Go 1.22
		funcs = append(funcs, func() int { return i })
	}
	var got []int
	for _, f := range funcs {
		got = append(got, f())
	}
	fmt.Println(got) // со старой семантикой было бы [4 4 4]
	// Output: [1 2 3]
}

// Метки позволяют выйти из вложенного цикла.
func Example_labels() {
	grid := [][]int{{1, 2}, {3, -1}, {5, 6}}
outer:
	for i, row := range grid {
		for j, v := range row {
			if v < 0 {
				fmt.Println("отрицательное в", i, j)
				break outer
			}
		}
	}
	// Output: отрицательное в 1 1
}

// Целочисленное деление, переполнение и преобразования типов.
func Example_numbers() {
	fmt.Println(7/2, -7/2, 7%3, -7%3, 7.0/2)
	var b uint8 = 255
	b++ // переполнение беззнаковых — по модулю, без паники
	fmt.Println(b)
	f := 3.99
	fmt.Println(int(f)) // отбрасывание дробной части, не округление
	// Output:
	// 3 -3 1 -1 3.5
	// 0
	// 3
}
