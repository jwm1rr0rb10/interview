// Пакет basics — функции, константы и управляющие конструкции.
package basics

import (
	"errors"
	"fmt"
)

// ErrDivByZero — ошибка деления на ноль.
var ErrDivByZero = errors.New("деление на ноль")

// DivMod возвращает частное и остаток: функции в Go могут возвращать
// несколько значений, и ошибка по соглашению идёт последней.
func DivMod(a, b int) (q, r int, err error) {
	if b == 0 {
		return 0, 0, ErrDivByZero
	}
	return a / b, a % b, nil
}

// Sum — вариадическая функция: nums внутри имеет тип []int.
func Sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

// NewCounter возвращает замыкание: переменная n живёт, пока живёт функция.
func NewCounter() func() int {
	n := 0
	return func() int {
		n++
		return n
	}
}

// Weekday — свой тип на базе int с константами через iota.
type Weekday int

const (
	Monday  Weekday = iota + 1 // 1
	Tuesday                    // 2: выражение повторяется, iota растёт
	Wednesday
	Thursday
	Friday
	Saturday
	Sunday
)

// IsWeekend — метод на своём типе.
func (d Weekday) IsWeekend() bool { return d == Saturday || d == Sunday }

// String делает тип fmt.Stringer: fmt вызовет его при печати.
func (d Weekday) String() string {
	names := [...]string{"?", "пн", "вт", "ср", "чт", "пт", "сб", "вс"}
	if d < Monday || d > Sunday {
		return fmt.Sprintf("Weekday(%d)", int(d))
	}
	return names[d]
}

// Grade показывает switch без условия — замену цепочке if-else.
func Grade(score int) string {
	switch {
	case score >= 90:
		return "отлично"
	case score >= 75:
		return "хорошо"
	case score >= 60:
		return "удовлетворительно"
	default:
		return "неудовлетворительно"
	}
}
