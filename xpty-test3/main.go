package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/ActiveState/cli/pkg/xpty"
)

func main() {
	// c := exec.Command("./test.sh")
	// c := exec.Command("/bin/sh", "-c", "echo")
	// c := exec.Command(".\\build\\state.exe", "auth", "signup")
	c := exec.Command(".\\demo.exe")
	// ptm, pts, err := pty.Open()
	xp, err := xpty.Open(130, 20)
	if err != nil {
		log.Fatalf("Failed to make Xpty: %v\n", err)
	}
	defer xp.Close()
	err = xp.StartProcessInTerminal(c)
	if err != nil {
		log.Fatalf("Failed to run command: %v\n", err)
	}

	/*ptm2, pts2, err := pty.Open()
	if err != nil {
		log.Fatalf("Failed to open ptm2")
	}
	defer ptm2.Close()
	*/

	f, err := os.OpenFile("testlogfile", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()

	fmt.Printf("created xp\n")
	time.Sleep(1 * time.Second)
	n, err := io.WriteString(xp.TerminalInPipe(), "abc\n")
	if err != nil {
		log.Fatalf("Failed to print username")
	}
	fmt.Printf("Send username (%d bytes)", n)
	time.Sleep(1 * time.Second)
	n, err = io.WriteString(xp.TerminalInPipe(), "abc\n")
	if err != nil {
		log.Fatalf("Failed to print password")
	}
	fmt.Printf("Send password1 (%d bytes)", n)
	time.Sleep(1 * time.Second)
	n, err = io.WriteString(xp.TerminalInPipe(), "incorrect\n")
	if err != nil {
		log.Fatalf("Failed to print password")
	}
	fmt.Printf("Send password2 (%d bytes)", n)

	go func() {
		io.Copy(f, xp.TerminalOutPipe())
	}()

	// time.Sleep(5 * time.Second)

	c.Wait()

	exitCode := c.ProcessState.ExitCode()
	if err != nil {
		log.Printf("Could not get exit code: %v\n", err)
	}
	log.Printf("exit code: %d", exitCode)

	fmt.Printf("state: %s", xp.State.String())

	return
}
