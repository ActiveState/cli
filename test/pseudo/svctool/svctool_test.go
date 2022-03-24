package svctool

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	simultaneous := 2
	iterations := 128
	pause := time.Millisecond * 10

	errs := make(chan error)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*300)
	defer cancel()

	go func() { // load up command instances
		defer close(errs)

		wg := &sync.WaitGroup{}
		var count int

		for iter := 0; iter < iterations; iter++ {
			start := make(chan struct{})
			wg.Add(simultaneous)
			for i := 0; i < simultaneous; i++ {
				count++
				fmt.Println("count", count)
				var ext string
				if runtime.GOOS == "windows" {
					ext = ".exe"
				}
				c := exec.CommandContext(ctx, filepath.Clean("../cmd/svc/build/svc"+ext))
				c.Stdout = os.Stdout

				go func(cmd *exec.Cmd) {
					defer wg.Done()

					select { // wait for common start time with fallback timeout
					case <-start:
					case <-time.After(time.Millisecond * 3):
						t.Error("cmd start is not aligned")
					}

					fmt.Println("STARTING")
					if err := cmd.Start(); err != nil {
						fmt.Println("start error:", err)
						errs <- err
					}

					if err := cmd.Wait(); err != nil {
						fmt.Println("wait error:", err)
						errs <- err
					}
				}(c)
			}

			close(start) // trip command instances executions
			time.Sleep(pause)
		}
		wg.Wait()
	}()

	var saved error
	var got int
	want := (simultaneous * iterations) - 1

	for err := range errs {
		var eerr *exec.ExitError
		if errors.As(err, &eerr) && eerr.ExitCode() < 0 { // ctx cancelled (ignore killed)
			continue
		}

		if saved == nil && err != nil { // save first non-nil error for example output
			saved = err
		}
		got++
		if got >= want {
			cancel()
		}
	}

	if got != want {
		t.Errorf("error count: got %d, want %d: %v", got, want, saved)
	}
}
