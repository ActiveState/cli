package main

import (
	"os"
	"os/exec"
)

func main() {
	cmd := exec.Command("bash", "--rcfile", "bashrc")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Start()
	if err != nil {
		panic(err.Error())
	}
	err = cmd.Wait()
	if err != nil {
		panic(err.Error())
	}
}
