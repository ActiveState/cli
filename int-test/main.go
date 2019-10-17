package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	expect "github.com/Netflix/go-expect"
	"github.com/hinshun/vt10x"
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
	f, err := os.OpenFile("testlogfile", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	f2, err := os.OpenFile("testprogress", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	c, state, err := vt10x.NewVT10XConsole(expect.WithStdout(os.Stdout), expect.WithLogger(log.New(f, "logger", 0)))
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command("./build/state", "auth")
	cmd.Stdin = c.Tty()
	cmd.Stdout = c.Tty()
	cmd.Stderr = c.Tty()

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	//time.Sleep(time.Second)
	f2.WriteString("searching username\n")
	c.Expect(expect.String("username:"))
	/*
		go func() {
			c.Expect(WithTimeoutMatcher(2 * time.Second))
		}()
	*/
	// time.Sleep(2 * time.Second)
	/*
		tChan := make(chan struct{}, 0)
		go func() {
			tChan <- struct{}{}
		}()
		<-tChan
	*/
	f2.WriteString("found username\n")
	c.Send("abc\n")
	// c.Send(fmt.Sprintf("\x1b[%dE", 1))
	f2.WriteString("wrote password\n")
	c.ExpectString("password:")
	f2.WriteString("sending password\n")
	// time.Sleep(time.Second)
	c.Send("def\n")
	f2.WriteString("sent password\n")
	c.SendLine("")
	c.Close()

	err = cmd.Wait()

	fmt.Printf("state: %v", state.String())
	f2.WriteString("stopped waiting for command\n")
	if err != nil {
		log.Fatal(err)
	}
}
