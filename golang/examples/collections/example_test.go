package collections

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode/utf8"
)

// Массив — значение фиксированной длины: присваивание копирует всё.
func Example_arrays() {
	a := [3]int{1, 2, 3}
	b := a // полная копия
	b[0] = 100
	fmt.Println(a, b, len(a))
	// Output: [1 2 3] [100 2 3] 3
}

// Срез — окно в массив: указатель, длина и ёмкость.
func Example_sliceBasics() {
	arr := [5]int{0, 1, 2, 3, 4}
	s := arr[1:3] // элементы 1 и 2; ёмкость — до конца массива
	fmt.Println(s, len(s), cap(s))

	s[0] = 100 // меняем общий массив
	fmt.Println(arr)

	var empty []int // nil-срез: len 0, append работает
	fmt.Println(empty == nil, len(empty))
	empty = append(empty, 1)
	fmt.Println(empty)
	// Output:
	// [1 2] 2 4
	// [0 100 2 3 4]
	// true 0
	// [1]
}

// Главная ловушка срезов: append пишет в общий массив, пока хватает ёмкости.
func Example_appendAliasing() {
	base := make([]int, 3, 10) // len 3, cap 10
	a := append(base, 1)       // места хватает: пишем в массив base
	b := append(base, 2)       // ...и затираем то, что записал a
	fmt.Println(a, b)

	// Решение 1: ограничить ёмкость полным выражением среза s[low:high:max].
	c := append(base[:3:3], 1) // cap == len, append выделит новый массив
	d := append(base[:3:3], 2)
	fmt.Println(c, d)

	// Решение 2: явная копия.
	e := append(slices.Clone(base), 3)
	fmt.Println(e)
	// Output:
	// [0 0 0 2] [0 0 0 2]
	// [0 0 0 1] [0 0 0 2]
	// [0 0 0 3]
}

// Когда ёмкости не хватает, append выделяет новый массив и копирует данные.
func Example_appendGrowth() {
	var s []int
	prevCap := -1
	for i := range 20 {
		s = append(s, i)
		if cap(s) != prevCap {
			fmt.Print(cap(s), " ")
			prevCap = cap(s)
		}
	}
	fmt.Println()
	// Output: 1 2 4 8 16 32
}

// Пакет slices покрывает почти все повседневные операции.
func Example_slicesPackage() {
	s := []int{5, 2, 8, 2, 9}
	fmt.Println(slices.Contains(s, 8), slices.Index(s, 2), slices.Max(s))

	sorted := slices.Sorted(slices.Values(s)) // новый отсортированный срез
	fmt.Println(sorted, s)

	s = slices.Insert(s, 1, 7) // вставить 7 на позицию 1
	s = slices.Delete(s, 3, 4) // удалить элемент с индексом 3
	fmt.Println(s, slices.Compact(slices.Sorted(slices.Values(s))))
	// Output:
	// true 1 9
	// [2 2 5 8 9] [5 2 8 2 9]
	// [5 7 2 2 9] [2 5 7 9]
}

// copy копирует min(len(dst), len(src)) элементов.
func Example_copy() {
	src := []int{1, 2, 3, 4}
	dst := make([]int, 2)
	n := copy(dst, src)
	fmt.Println(n, dst)
	// Output: 2 [1 2]
}

// Map: чтение отсутствующего ключа даёт нулевое значение.
func Example_maps() {
	ages := map[string]int{"аня": 31, "боря": 25}
	fmt.Println(ages["вика"]) // 0, а не ошибка

	if age, ok := ages["вика"]; !ok { // «comma ok» отличает 0 от отсутствия
		fmt.Println("вики нет", age)
	}
	ages["вика"] = 28
	delete(ages, "боря")
	fmt.Println(len(ages))

	// Порядок обхода map случаен — для стабильного вывода сортируем ключи.
	for _, name := range slices.Sorted(maps.Keys(ages)) {
		fmt.Println(name, ages[name])
	}

	var nilMap map[string]int
	fmt.Println(nilMap["x"], len(nilMap)) // читать nil-map можно, писать — panic
	// Output:
	// 0
	// вики нет 0
	// 2
	// аня 31
	// вика 28
	// 0 0
}

// Множество — map с пустой структурой в значении: она не занимает памяти.
func Example_set() {
	seen := make(map[string]struct{})
	for _, w := range []string{"a", "b", "a", "c", "b"} {
		seen[w] = struct{}{}
	}
	fmt.Println(len(seen), slices.Sorted(maps.Keys(seen)))
	// Output: 3 [a b c]
}

// Строка — неизменяемые байты UTF-8. len считает байты, range — руны.
func Example_strings() {
	s := "Привет, Go"
	fmt.Println(len(s), utf8.RuneCountInString(s))
	fmt.Println(s[0], string(s[0:2])) // индекс даёт байт, срез — подстроку байт

	var parts []string
	for i, r := range "Gö!" {
		parts = append(parts, fmt.Sprintf("%d:%c", i, r)) // i — смещение в байтах
	}
	fmt.Println(strings.Join(parts, " "))

	runes := []rune(s)
	fmt.Println(string(runes[:6]))
	// Output:
	// 16 10
	// 208 П
	// 0:G 1:ö 3:!
	// Привет
}

// Склеивать строки в цикле — через strings.Builder, а не +=.
func Example_builder() {
	var b strings.Builder
	for i := range 3 {
		fmt.Fprintf(&b, "[%d]", i)
	}
	fmt.Println(b.String())
	// Output: [0][1][2]
}

// Пакет strings: Cut, CutLast (Go 1.27), Fields, итераторы.
func Example_stringsPackage() {
	key, value, ok := strings.Cut("host=localhost", "=")
	fmt.Println(key, value, ok)

	dir, file, _ := strings.CutLast("/var/log/app.log", "/")
	fmt.Println(dir, file)

	for part := range strings.SplitSeq("a,b,c", ",") { // без промежуточного среза
		fmt.Print("<", part, ">")
	}
	fmt.Println()
	fmt.Println(strings.EqualFold("Go", "GO"), strings.Repeat("ab", 3))
	// Output:
	// host localhost true
	// /var/log app.log
	// <a><b><c>
	// true ababab
}

// Структуры, анонимные структуры и сравнение.
func Example_structs() {
	type Point struct{ X, Y int }
	p := Point{X: 1, Y: 2} // всегда указывай имена полей
	q := p                 // копия
	q.X = 10
	fmt.Println(p, q, p == Point{1, 2})

	pp := &p
	pp.Y = 20 // автоматическое разыменование: (*pp).Y
	fmt.Printf("%+v\n", p)

	cfg := struct {
		Host string
		Port int
	}{"localhost", 8080}
	fmt.Println(cfg.Host, cfg.Port)
	// Output:
	// {1 2} {10 2} true
	// {X:1 Y:20}
	// localhost 8080
}

// Map со структурами в значении: поле нельзя менять на месте.
func Example_mapOfStructs() {
	type User struct{ Visits int }
	users := map[string]User{"аня": {}}
	// users["аня"].Visits++ — ошибка компиляции: значение map не адресуемо
	u := users["аня"]
	u.Visits++
	users["аня"] = u // прочитать, изменить, записать обратно

	ptrs := map[string]*User{"боря": {}}
	ptrs["боря"].Visits++ // с указателями можно, но это общий объект
	fmt.Println(users["аня"].Visits, ptrs["боря"].Visits)
	// Output: 1 1
}

// clear обнуляет map или элементы среза (Go 1.21).
func Example_clear() {
	m := map[int]bool{1: true, 2: true}
	s := []int{1, 2, 3}
	clear(m)
	clear(s)
	fmt.Println(len(m), s, len(s))
	// Output: 0 [0 0 0] 3
}
