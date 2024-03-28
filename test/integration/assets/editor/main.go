package main

import (
	"fmt"
	"os"
)

func main() {
	file := os.Args[1]
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	defer f.Close()
	if _, err := f.WriteString("\nmore info!"); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
