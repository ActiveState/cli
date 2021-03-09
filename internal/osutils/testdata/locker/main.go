package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ActiveState/cli/internal/osutils/lockfile"
)

func main() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(2*time.Minute))
	defer cancel()
	go func() {
		select {
		case <-c:
		case <-ctx.Done():
		}
		cancel()
	}()

	if len(os.Args) < 3 {
		fmt.Printf("two arguments required")
		os.Exit(2)
	}
	keep := os.Args[2] == "keep"
	pl, err := lockfile.NewPidLock(os.Args[1])
	if err != nil {
		log.Fatalf("Could not open lock file: %s", os.Args[1])
	}
	ok, _ := pl.TryLock()
	if !ok {
		fmt.Println("DENIED")
		return
	}

	fmt.Println("LOCKED")
	if keep {
		pl.Close(keep)
	}

	<-ctx.Done()
	fmt.Println("done")

	pl.Close()
}
