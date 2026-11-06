package interfaces

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func ExampleTotalArea() {
	r := Rect{W: 2, H: 3}
	r.Scale(2) // Go сам возьмёт &r: r адресуема
	fmt.Println(r, r.Area())
	fmt.Printf("%.2f\n", TotalArea(r, Circle{R: 1}))
	// Output:
	// {4 6} 24
	// 27.14
}

func ExampleDescribe() {
	fmt.Println(Describe(Rect{W: 1, H: 2}))
	fmt.Println(Describe(Circle{R: 3}))
	fmt.Println(Describe(errors.New("сбой")))
	fmt.Println(Describe(42))
	fmt.Println(Describe(3.14))
	fmt.Println(Describe(nil))
	// Output:
	// прямоугольник 1x2
	// stringer: Circle(r=3)
	// ошибка: сбой
	// целое 42
	// неизвестно: float64
	// nil
}

// Утверждение типа с проверкой «comma ok» не паникует.
func Example_typeAssertion() {
	var s Shape = Circle{R: 2}
	if c, ok := s.(Circle); ok {
		fmt.Println("круг радиуса", c.R)
	}
	_, ok := s.(Rect)
	fmt.Println(ok)
	// Проверка на дополнительное поведение: умеет ли фигура String()?
	if st, ok := s.(fmt.Stringer); ok {
		fmt.Println(st)
	}
	// Output:
	// круг радиуса 2
	// false
	// Circle(r=2)
}

// Метод с получателем-указателем не входит в набор методов значения.
func Example_methodSets() {
	type Scaler interface{ Scale(float64) }
	var _ Scaler = &Rect{} // *Rect имеет Scale
	// var _ Scaler = Rect{} — не компилируется: у Rect (значения) нет Scale
	fmt.Println("ok")
	// Output: ok
}

func ExampleEmployee() {
	// До Go 1.27 нужно было писать Employee{Base: Base{ID: 7}, ...};
	// теперь поле встроенной структуры можно указать прямо в литерале.
	e := Employee{ID: 7, Name: "Аня"}
	fmt.Println(e.ID, e.Base.ID, e.Describe()) // поле и метод Base доступны напрямую
	// Output: 7 7 объект #7
}

func ExampleCountingWriter() {
	cw := &CountingWriter{Writer: os.Stdout}
	fmt.Fprintln(cw, "привет")                     // fmt пишет в любой io.Writer
	_, _ = io.Copy(cw, strings.NewReader("мир\n")) // и io.Copy тоже
	fmt.Println(cw.N, "байт")
	// Output:
	// привет
	// мир
	// 20 байт
}

type MyErr struct{}

func (*MyErr) Error() string { return "моя ошибка" }

// Главная ловушка интерфейсов: интерфейс с nil-указателем внутри — не nil.
func Example_nilInterface() {
	find := func(fail bool) error {
		var e *MyErr // nil-указатель
		if fail {
			e = &MyErr{}
		}
		return e // ОШИБКА: интерфейс получает тип *MyErr и значение nil
	}
	err := find(false)
	fmt.Println(err == nil) // false!
	fmt.Printf("%T %v\n", err, err == (*MyErr)(nil))

	findOK := func(fail bool) error {
		if fail {
			return &MyErr{}
		}
		return nil // правильно: возвращать nil явно
	}
	fmt.Println(findOK(false) == nil)
	// Output:
	// false
	// *interfaces.MyErr true
	// true
}
