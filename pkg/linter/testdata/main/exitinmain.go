// Пакет для теста линтера с вызовом os.Exit из main.
package main

import (
	"os"
)

// Функция main в которой вызывается os.Exit .
func main() {
	println("Hello and goodby world")
	os.Exit(1) // want `Exit called in main package`
}
