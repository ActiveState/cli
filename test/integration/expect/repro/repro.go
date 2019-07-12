package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/kr/pty"
)

type Process struct {
	cmd   *exec.Cmd
	pty   *os.File
	stdin io.Writer
}

func (p *Process) start() error {
	//_, outWriter := io.Pipe()
	stdinR, stdinW := io.Pipe()

	outWriter := &stdWriter{}
	p.stdin = stdinW

	var err error
	if p.pty, err = pty.Start(p.cmd); err != nil {
		return err
	}

	go func() {
		_, err := io.Copy(outWriter, p.pty)
		if err != nil {
			log.Printf("Error while copying pty to output pipe: %v", err)
		}
	}()
	go func() {
		defer stdinW.Close()
		_, err = io.Copy(p.pty, stdinR)
		if err != nil {
			log.Printf("Error while copying input pipe to pty: %v", err)
		}
	}()

	return nil
}

type stdWriter struct {
}

func (w *stdWriter) Write(p []byte) (n int, err error) {
	fmt.Printf("Writing: %s", string(p))
	return len(p), nil
}

func main() {
	p := &Process{
		cmd: exec.Command("./sub/sub"),
	}
	p.start()

	go func() {
		fmt.Fprintf(p.stdin, "tty\n")
		fmt.Fprintf(p.stdin, "echo \"-- OUT -- $ACTIVESTATE_ACTIVATED -- OUT --\"\n")
		fmt.Fprintf(p.stdin, "exit\n")
	}()
	p.cmd.Wait()
}
