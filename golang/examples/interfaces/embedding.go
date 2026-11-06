package interfaces

import (
	"fmt"
	"io"
)

// Встраивание: поля и методы Base становятся доступны через Employee.
// Это композиция, а не наследование: Employee не является Base.
type Base struct {
	ID int
}

func (b Base) Describe() string { return fmt.Sprintf("объект #%d", b.ID) }

type Employee struct {
	Base // встроенное поле: имя поля — имя типа
	Name string
}

// CountingWriter — декоратор: оборачивает любой io.Writer и считает байты.
// Встроенный интерфейс даёт все его методы «бесплатно», а Write мы переопределяем.
type CountingWriter struct {
	io.Writer
	N int64
}

func (cw *CountingWriter) Write(p []byte) (int, error) {
	n, err := cw.Writer.Write(p) // явный вызов «родительского» метода
	cw.N += int64(n)
	return n, err
}

// ReadWriter показывает встраивание интерфейсов друг в друга.
type ReadWriter interface {
	io.Reader
	io.Writer
}
