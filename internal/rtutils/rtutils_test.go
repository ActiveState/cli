package rtutils

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	assert.True(t, strings.HasPrefix(err.Error(), closerErr.Error()))
	assert.True(t, errors.Is(err, returnErr))

	err = func() (rerr error) {
		defer Closer(closer, &rerr)
		return nil
	}()
	require.Error(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), closerErr.Error()))
	assert.False(t, errors.Is(err, returnErr))
}

func TestTimeout(t *testing.T) {
	err := Timeout(func() error {
		time.Sleep(time.Millisecond * 20)
		return nil
	}, time.Millisecond*10)
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