// Пакет interfaces — методы, интерфейсы и встраивание.
package interfaces

import (
	"fmt"
	"math"
)

// Shape — маленький интерфейс: тип реализует его неявно,
// просто имея нужные методы. Ключевого слова implements нет.
type Shape interface {
	Area() float64
	Perimeter() float64
}

type Rect struct{ W, H float64 }

// Методы с получателем-значением: работают с копией, Rect не меняют.
func (r Rect) Area() float64      { return r.W * r.H }
func (r Rect) Perimeter() float64 { return 2 * (r.W + r.H) }

// Scale меняет Rect, поэтому получатель — указатель.
func (r *Rect) Scale(k float64) {
	r.W *= k
	r.H *= k
}

type Circle struct{ R float64 }

func (c Circle) Area() float64      { return math.Pi * c.R * c.R }
func (c Circle) Perimeter() float64 { return 2 * math.Pi * c.R }
func (c Circle) String() string     { return fmt.Sprintf("Circle(r=%g)", c.R) }

// Проверка при компиляции: если Rect перестанет реализовывать Shape,
// сборка упадёт здесь, а не где-то в месте использования.
var (
	_ Shape = Rect{}
	_ Shape = Circle{}
)

// TotalArea принимает интерфейс: ей всё равно, какие фигуры внутри.
func TotalArea(shapes ...Shape) float64 {
	total := 0.0
	for _, s := range shapes {
		total += s.Area()
	}
	return total
}

// Describe показывает type switch: ветвление по динамическому типу.
func Describe(v any) string {
	switch x := v.(type) {
	case nil:
		return "nil"
	case Rect:
		return fmt.Sprintf("прямоугольник %gx%g", x.W, x.H)
	case fmt.Stringer: // любой тип с методом String()
		return "stringer: " + x.String()
	case error:
		return "ошибка: " + x.Error()
	case int, int64:
		return fmt.Sprintf("целое %v", x) // в ветке с несколькими типами x имеет тип any
	default:
		return fmt.Sprintf("неизвестно: %T", x)
	}
}
