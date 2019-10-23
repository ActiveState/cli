package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"syscall"
	"time"

	"github.com/ActiveState/cli/int-test/conpty"
	expect "github.com/Netflix/go-expect"
)

type TimeoutMatcher struct {
	expireTime time.Time
}

func (tom *TimeoutMatcher) Match(buf *bytes.Buffer) bool {
	return (time.Now().UnixNano() > tom.expireTime.UnixNano())
}

func WithTimeoutMatcher(d time.Duration) expect.ExpectOpt {
	return func(eo *expect.ExpectOpts) error {
		eo.Matchers = append(eo.Matchers, &TimeoutMatcher{time.Now().Add(d)})
		return nil
	}
}

func main() {
	wpty := conpty.New()
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
	go func() {
		fmt.Println("reading from stdout")
		b := make([]byte, 500)
		// n, err := wpty.ReadStdout(b)
		n, err := wpty.PipeOut.Read(b)
		if err != nil {
			fmt.Printf("Failed reading from pipe: %v\n", err)
		}
		fmt.Printf("read: %d bytes: %s\n", n, string(b[:n]))
		f.WriteString(string(b[:n]))
	}()
	go func() {
		// give it one second to get ready for input
		time.Sleep(time.Second)
		_, err := wpty.PipeIn.WriteString("Hello world\n")
		if err != nil {
			fmt.Printf("Failed writing to pipe: %v\n", err)
		}
		fmt.Printf("wrote to pipe...")
		b := make([]byte, 500)
		// n, err := wpty.ReadStdout(b)
		n, err := wpty.PipeOut.Read(b)
		if err != nil {
			fmt.Printf("Failed reading from pipe: %v\n", err)
		}
		fmt.Printf("read: %d bytes: %s\n", n, string(b[:n]))
		f.WriteString(string(b[:n]))
	}()
	time.Sleep(2 * time.Second)

	return
}
