package rtutils

import (
	"errors"
	"fmt"
	"strings"
	"testing"

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
