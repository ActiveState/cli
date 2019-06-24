package main

import "time"

func main() {
	for i := 0; i < 10; i++ {
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
