package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ActiveState/cli/internal/osutils"
)

func main() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	if len(os.Args) < 3 {
		fmt.Printf("two arguments required")
		os.Exit(2)
	}
	keep := os.Args[2] == "keep"
	pl, err := osutils.NewPidLock(os.Args[1])
	if err != nil {
		log.Fatalf("Could not open lock file: %s", os.Args[1])
	}
	ok, _ := pl.TryLock()
	if !ok {
		fmt.Printf("DENIED")
	}

	fmt.Println("LOCKED")
	if keep {
		pl.Close(keep)
	}

	select {
	case <-c:
	case <-time.After(1 * time.Hour):
	}

	fmt.Println("done")

	defer pl.Close()
}
