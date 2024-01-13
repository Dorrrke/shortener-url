package main

import "os"

func main() {
	println("Hello world")
	t := Exiter{}
	t.Exit(1)
}

type Exiter struct {
}

func (e Exiter) Exit(code int) {
	os.Exit(code)
}
