package svctool

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	count := 128

	start := make(chan struct{})
	errs := make(chan error)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	go func() { // load up command instances
		defer close(errs)

		wg := &sync.WaitGroup{}
		wg.Add(count)

		for i := 0; i < count; i++ {
			c := exec.CommandContext(ctx, filepath.Clean("../../cmd/svc/build/svc"))
			buf := &bytes.Buffer{}
			c.Stdout = buf

			go func(buf *bytes.Buffer, cmd *exec.Cmd) {
				defer wg.Done()
				defer func() {
					//fmt.Println(buf.String())
				}()

				select { // wait for common start time with fallback timeout
				case <-start:
				case <-time.After(time.Millisecond * 3):
					t.Error("cmd start is not aligned")
				}

				if err := cmd.Start(); err != nil {
					errs <- err
				}

				if err := cmd.Wait(); err != nil {
					errs <- err
				}
			}(buf, c)
		}
		wg.Wait()
	}()

	close(start) // trip command instances executions

	var saved error
	var got int
	want := count - 1

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
