package main

import "strings"

// Greeting возвращает приветствие. Логика вынесена из main,
// чтобы её можно было протестировать.
func Greeting(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "мир"
	}
	return "Привет, " + name + "!"
}
