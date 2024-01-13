package main

import (
	"os"
)

func main() {
	println("Hello and goodby world")
	os.Exit(1) // want `Exit called in main package`
}
