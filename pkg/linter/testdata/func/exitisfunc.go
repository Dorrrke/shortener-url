package main

import (
	"os"
)

func main() {
	println("Nice joke")
	Exit(1)
}

func Exit(code int) {
	os.Exit(code)
}
