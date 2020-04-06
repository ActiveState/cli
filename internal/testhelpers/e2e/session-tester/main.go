package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

var exit1 = flag.Bool("exit1", false, "exit the script with exit code 1")
var sleep = flag.Bool("sleep", false, "sleep for an hour, basically never return unless interrupted")
var fillBuffer = flag.Bool("fill-buffer", false, "print a string with 1,000,000 characters")
var stutter = flag.Bool("stutter", false, "print 5 messages with 50 ms delays")

func main() {
	flag.Parse()

	fmt.Println("an expected string")

	if *sleep {
		time.Sleep(1 * time.Hour)
	}

	if *fillBuffer {
		for i := 0; i < 1e6; i++ {
			os.Stdout.Write([]byte("a"))
		}
		os.Stdout.Write([]byte("\n"))
	}

	if *stutter {
		for i := 0; i < 5; i++ {
			fmt.Printf("stuttered %d times\n", i+1)
			time.Sleep(50 * time.Millisecond)
		}
	}

	if *exit1 {
		os.Exit(1)
	}
}
