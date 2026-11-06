package main

import "testing"

func TestGreeting(t *testing.T) {
	if got, want := Greeting("Гофер"), "Привет, Гофер!"; got != want {
		t.Errorf("Greeting(%q) = %q, want %q", "Гофер", got, want)
	}
	if got, want := Greeting("  "), "Привет, мир!"; got != want {
		t.Errorf("Greeting(пробелы) = %q, want %q", got, want)
	}
}
