package conc

import (
	"context"
	"fmt"
	"slices"
	"time"
)

func Example_pipeline() {
	ctx := context.Background()
	for v := range Square(ctx, Generate(ctx, 1, 2, 3, 4)) {
		fmt.Print(v, ";")
	}
	fmt.Println()
	// Output: 1;4;9;16;
}

func ExampleMerge() {
	ctx := context.Background()
	merged := Merge(ctx, Generate(ctx, 1, 2, 3), Generate(ctx, 10, 20))
	var got []int
	for v := range merged {
		got = append(got, v)
	}
	slices.Sort(got) // порядок из разных горутин не определён
	fmt.Println(got)
	// Output: [1 2 3 10 20]
}

// Отмена контекста останавливает конвейер, даже если данные ещё есть.
func Example_cancel() {
	ctx, cancel := context.WithCancel(context.Background())
	nums := Generate(ctx, 1, 2, 3, 4, 5)
	fmt.Println(<-nums, <-nums)
	cancel()
	time.Sleep(10 * time.Millisecond) // дать горутине увидеть отмену
	count := 0
	for range nums { // канал закроется, цикл завершится
		count++
	}
	fmt.Println(count <= 1) // максимум одно значение уже было в пути
	// Output:
	// 1 2
	// true
}

func ExampleFirst() {
	fast := func() string { return "быстрый" }
	slow := func() string { time.Sleep(time.Second); return "медленный" }
	r, err := First(500*time.Millisecond, slow, fast)
	fmt.Println(r, err)
	_, err = First(10*time.Millisecond, slow)
	fmt.Println(err)
	// Output:
	// быстрый <nil>
	// timeout
}

// Небуферизованный канал — точка синхронизации: отправитель ждёт получателя.
func Example_unbuffered() {
	done := make(chan struct{})
	go func() {
		fmt.Println("работа в горутине")
		close(done) // сигнал всем получателям сразу
	}()
	<-done
	fmt.Println("main дождался")
	// Output:
	// работа в горутине
	// main дождался
}

// Чтение из закрытого канала не блокируется: нулевое значение и ok == false.
func Example_closedChannel() {
	ch := make(chan int, 2)
	ch <- 7
	close(ch)
	v, ok := <-ch
	fmt.Println(v, ok)
	v, ok = <-ch
	fmt.Println(v, ok)
	// Output:
	// 7 true
	// 0 false
}

// nil-канал блокирует навсегда — так в select «выключают» ветку.
func Example_nilChannelInSelect() {
	a := make(chan int)
	b := make(chan int)
	go func() { a <- 1; close(a) }()
	go func() { b <- 2; close(b) }()
	sum := 0
	for a != nil || b != nil {
		select {
		case v, ok := <-a:
			if !ok {
				a = nil // ветка больше не выбирается
				continue
			}
			sum += v
		case v, ok := <-b:
			if !ok {
				b = nil
				continue
			}
			sum += v
		}
	}
	fmt.Println(sum)
	// Output: 3
}
