package svctool

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

//go:generate go build -o ../cmd/svc/build/svc ../cmd/svc
//go:generate go build -o ../cmd/svc/build/tool ../cmd/tool

type logFunc func(...interface{})

func (l logFunc) Write(p []byte) (int, error) {
	l(string(p))
	return len(p), nil
}

func TestService(t *testing.T) {
	simultaneous := 2
	iterations := 512
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
				t.Log("count", count)
				var ext string
				if runtime.GOOS == "windows" {
					ext = ".exe"
				}
				c := exec.CommandContext(ctx, filepath.Clean("../cmd/svc/build/svc"+ext))
				c.Stdout = logFunc(t.Log)

				go func(cmd *exec.Cmd) {
					defer wg.Done()

					select { // wait for common start time with fallback timeout
					case <-start:
					case <-time.After(time.Millisecond * 3):
						t.Error("cmd start is not aligned")
					}

					t.Log("STARTING")
					if err := cmd.Start(); err != nil {
						t.Log("start error:", err)
						errs <- err
					}

					if err := cmd.Wait(); err != nil {
						t.Log("wait error:", err)
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
