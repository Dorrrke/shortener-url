package main

import "os"

func main() {
	println("here is it")
	f := func(code int) { os.Exit(code) } // want `Exit called in main package`
	f(1)
}
