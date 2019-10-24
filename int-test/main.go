package main

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"time"

	"github.com/ActiveState/cli/int-test/conpty"
	"github.com/hinshun/vt10x"
)

func main() {
	wpty := conpty.New(80, 40)
	var state vt10x.State
	stateLog, _ := os.Create("state.log")
	defer stateLog.Close()
	l := log.New(stateLog, "state: ", 0)
	l.Printf("test")
	defer wpty.Close()
	err := wpty.CreatePseudoConsoleAndPipes()
	if err != nil {
		log.Fatalf("Could not create pseudo terminal: %v", err)
	}
	err = wpty.InitializeStartupInfoAttachedToPTY()
	if err != nil {
		log.Fatalf("could not initialize extended startup info: %v", err)
	}
	_, p, err := wpty.Spawn([]string{".\\test.bat"})
	if err != nil {
		log.Fatalf("windows error: %v", err)
	}
	var exitCode uint32
	err = syscall.GetExitCodeProcess(syscall.Handle(p), &exitCode)
	if err != nil {
		log.Printf("Could not get exit code: %v\n", err)
	}
	log.Printf("exit code: %d", exitCode)

	term, err := vt10x.Create(&state, wpty.PipeOut)
	if err != nil {
		log.Fatalf("Could not create vt10x terminal: %v", err)
	}
	defer term.Close()
	state.DebugLogger = l
	term.Resize(80, 40)

	go func() {
		for {
			err := term.Parse()
			fmt.Printf("parsed")
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				break
			}
		}
	}()

	fmt.Printf("create wpty\n")
	// wpty.PipeIn.WriteString("abc")
	// fmt.Printf("written the stuff\n")
	time.Sleep(2 * time.Second)
	/* b := make([]byte, 5)
	n, err := wpty.ReadStdout(b)
	// n, err := wpty.PipeOut.Read(b)
	if err != nil {
		fmt.Printf("Failed reading from pipe: %v\n", err)
	}

	fmt.Printf("read: %s\n", string(b[:n]))
	*/
	f, err := os.OpenFile("testlogfile", os.O_RDWR|os.O_CREATE, 0666)
	defer f.Close()
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	/*
		go func() {
			fmt.Println("reading from stdout")
			b := make([]byte, 2000)
			// n, err := wpty.ReadStdout(b)
			n, err := wpty.PipeOut.Read(b)
			if err != nil {
				fmt.Printf("Failed reading from pipe: %v\n", err)
			}
			fmt.Printf("read: %d bytes\n", n)
			f.WriteString(string(b[:n]))
		}()
		go func() {
			// give it one second to get ready for input
			time.Sleep(time.Second)
			_, err := wpty.PipeIn.WriteString("abcdefg world\r\n\n")
			if err != nil {
				fmt.Printf("Failed writing to pipe: %v\n", err)
			}
			fmt.Printf("wrote to pipe...")
			b := make([]byte, 2000)
			// n, err := wpty.ReadStdout(b)
			n, err := wpty.PipeOut.Read(b)
			if err != nil {
				fmt.Printf("Failed reading from pipe: %v\n", err)
			}
			fmt.Printf("read 2: %d bytes\n", n)
			f.WriteString(string(b[:n]))
		}()
	*/
	time.Sleep(2 * time.Second)
	fmt.Printf("state: %s", state.String())

	return
}
