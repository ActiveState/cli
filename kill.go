package main

import (
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/installation"
)

func main() {
	path := "/home/dev/cli/build"
	err := installation.StopRunning(path)
	if err != nil {
		fmt.Println("Could not stop running:", err)
		os.Exit(1)
	}
	fmt.Println("Stopped running")
}
