package sighandler

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/autarch/testify/assert"
	"github.com/autarch/testify/require"
)

func TestBackgroundSigHandler(t *testing.T) {
	called := make(chan bool)

	bs := NewBackgroundSignalHandler(func(s os.Signal) {
		called <- true
	}, os.Interrupt)
	Push(bs)
	assert.Lenf(t, stack.stack, 1, "signal stack should have one entry")
	defer func() {
		err := Pop()
		require.NoError(t, err)

		assert.Len(t, stack.stack, 0, "signal stack should be empty")
	}()

	// fake an interrupt signal
	bs.sigCh <- os.Interrupt

	// expect to receive a called event

	select {
	case v := <-called:
		if !v {
			t.Fatalf("Expected background task to send 'true'")
		}
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for callback")
	}
}

func TestAwaitingSigHandler(t *testing.T) {
	as := NewAwaitingSigHandler(os.Interrupt)
	Push(as)

	defer func() {
		err := Pop()
		require.NoError(t, err)

		assert.Len(t, stack.stack, 0, "signal stack should be empty")
	}()

	t.Run("without signal", func(t *testing.T) {
		err := as.WaitForFunc(func() error {
			return nil
		})
		require.NoError(t, err)
	})

	t.Run("return with signal error", func(t *testing.T) {
		go func() {
			as.sigCh <- os.Interrupt
		}()

		err := as.WaitForFunc(func() error {
			time.Sleep(time.Second)
			return nil
		})
		require.Error(t, err, "should have received error")
		var serr interface{ Signal() os.Signal }
		if !errors.As(err, &serr) {
			t.Fatalf("expected error to be a SignalError")
		}
	})

	t.Run("ignore signal error", func(t *testing.T) {
		go func() {
			time.Sleep(time.Second)
			as.sigCh <- os.Interrupt
		}()

		err := as.WaitForFunc(func() error {
			return nil
		})
		require.NoError(t, err, "should have received error")
	})
}
