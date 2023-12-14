package main

import (
	"fmt"
	"os"
)

func main() {
	env := os.Environ()
	for _, e := range env {
		fmt.Println(e)
	}
}
