// +build linux darwin

package expect

import (
	"fmt"
	"log"
	"os"

	ptyexpect "github.com/Netflix/go-expect"
	"github.com/hinshun/vt10x"
)

func (p *Process) start() error {
	fmt.Printf("Start the process")
	// redirect the inputs to the pseudo terminal

	// run the command
	err := p.cmd.Start()
	if err != nil {
		return fmt.Errorf("Error starting process: %v", err)
	}

	return nil
}

func (p *Process) setupPTY() error {

	var err error
	p.logfile, err = os.OpenFile("testlogfile2", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	p.logfile.WriteString("test")

	p.console, p.state, err = vt10x.NewVT10XConsole(
		ptyexpect.WithStdout(os.Stdout),
		ptyexpect.WithLogger(log.New(p.logfile, "logger", 0)),
	)
	if err != nil {
		return fmt.Errorf("Error spawning a pseudo terminal: %v", err)
	}
	return nil

}

func (p *Process) setupStdin() {
	p.cmd.Stdin = p.console.Tty()
}

func (p *Process) setupStdout() {
	p.cmd.Stdout = p.console.Tty()
}

func (p *Process) setupStderr() {
	p.cmd.Stderr = p.console.Tty()
}

func (p *Process) close() error {
	p.logfile.Close()
	err := p.console.Close()
	if err != nil {
		return fmt.Errorf("Error closing the pseudo terminal: %v", err)
	}
	return nil
}

func (p *Process) Write(input string) error {
	_, err := p.console.SendLine(input)
	return err
}
