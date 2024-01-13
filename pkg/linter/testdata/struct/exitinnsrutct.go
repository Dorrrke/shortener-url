// Пакет для теста линтера с вызовом os.Exit, находящийся в методе структуры, из main.
package main

import "os"

// Функция main в которой вызывается метод  Exit структуры Exiter.
func main() {
	println("Hello world")
	t := Exiter{}
	t.Exit(1)
}

// Тестовая структура.
type Exiter struct {
}

// Exit - метод тестовой структуры с вызовом os.Exit .
func (e Exiter) Exit(code int) {
	os.Exit(code)
}
