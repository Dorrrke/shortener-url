// Пакет для теста линтера с вызовом os.Exit, находящийся в функции находящейся в структуре, из main.
package main

import "os"

// Функция main в которой вызывается функция f.
func main() {
	println("here is it")
	f := func(code int) { os.Exit(code) } // want `Exit called in main package`
	f(1)
}
