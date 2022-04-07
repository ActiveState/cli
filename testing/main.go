package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Main thread")
	anotherFunc()
	for i := 0; i < 10; i++ {
		time.Sleep(time.Second)
		fmt.Printf("Still main thread: %d\n", i)
	}
}

func anotherFunc() {
	go func() {
		for i := 0; i < 5; i++ {
			fmt.Printf("Still working: %d\n", i)
			time.Sleep(time.Second)
		}
	}()

	fmt.Println("another func")
	return
}
