// Пакет для теста линтера с вызовом os.Exit, находящийся в отдельной функции, из main.
package main

import (
	"os"
)

// Функция main в которой вызывается функция в которой вызывается os.Exit .
func main() {
	println("Nice joke")
	Exit(1)
}

// Функция в которой вызывается os.Exit .
func Exit(code int) {
	os.Exit(code)
}
