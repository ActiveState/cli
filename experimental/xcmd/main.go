package main

import (
	"os"
	"time"
)

func main() {
	os.Setenv("CLITEST", "yeup")

	for i := 0; i < 2; i++ {
		cmd, err := newXCmd()
		check(err)

		go func() {
			time.Sleep(time.Second * 6)

			check(cmd.close())
		}()

		check(cmd.wait())
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
