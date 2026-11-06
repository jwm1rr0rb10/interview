package options

import (
	"fmt"
	"time"
)

func ExampleNew() {
	c, err := New("payments:443")
	fmt.Println(c.timeout, c.retries, err)

	c, err = New("payments:443", WithTimeout(2*time.Second), WithRetries(5))
	fmt.Println(c.timeout, c.retries, err)

	_, err = New("payments:443", WithTimeout(-1))
	fmt.Println(err)
	// Output:
	// 5s 3 <nil>
	// 2s 5 <nil>
	// timeout must be positive
}

func ExampleNewFromConfig() {
	c := NewFromConfig("payments:443", Config{Retries: 1})
	fmt.Println(c.timeout, c.retries)
	// Output: 5s 1
}
