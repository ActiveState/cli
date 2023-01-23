package rtutils

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_Closer(t *testing.T) {
	closerErr := fmt.Errorf("Closer error")
	returnErr := fmt.Errorf("Returned error")
	closer := func() error {
		return closerErr
	}
	err := func() (rerr error) {
		defer Closer(closer, &rerr)
		return returnErr
	}()

	require.Error(t, err)
	if !errors.Is(err, closerErr) {
		t.Errorf("Expected error to match closerErr: %v, got %v", closerErr, err)
	}
	if !errors.Is(err, returnErr) {
		t.Errorf("Expected error to match returnErr: %v, got %v", returnErr, err)
	}
}

func TestTimeout(t *testing.T) {
	err := Timeout(func() error {
		time.Sleep(time.Millisecond * 100)
		return nil
	}, time.Millisecond)
	require.True(t, errors.Is(err, ErrTimeout), "Should return timeout error, actual: %v", err)

	v := false
	err = Timeout(func() error {
		v = true
		return nil
	}, time.Millisecond*10)
	require.NoError(t, err)
	require.True(t, v, "Value should've been modified")

	expected := fmt.Errorf("I'm an error")
	err = Timeout(func() error {
		return expected
	}, time.Millisecond*10)
	require.Error(t, err)
	require.Equal(t, expected, err)
}
