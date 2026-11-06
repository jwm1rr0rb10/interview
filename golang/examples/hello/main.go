// Программа hello приветствует пользователя по имени из флага.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	name := flag.String("name", "", "кого приветствовать")
	flag.Parse()

	if len(*name) > 100 {
		fmt.Fprintln(os.Stderr, "слишком длинное имя")
		os.Exit(2) // ненулевой код выхода — сигнал ошибки для shell и CI
	}
	fmt.Println(Greeting(*name))
}
