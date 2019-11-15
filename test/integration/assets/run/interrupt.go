package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"
)

func main() {
	c := make(chan os.Signal, 1)
	defer close(c)
	signal.Notify(c, os.Interrupt)
	fmt.Println("Start of script")

	select {
	case <-time.After(time.Hour):
		fmt.Println("timed out, no signal received.")
	case sig := <-c:
		fmt.Printf("received %v\n", sig)
	}

	fmt.Println("After first sleep or interrupt")

	select {
	case <-time.After(2 * time.Second):
		fmt.Println("After second sleep")
	case sig := <-c:
		fmt.Printf("received %v\n", sig)
		os.Exit(123)
	}
}
