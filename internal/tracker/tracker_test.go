package tracker

import (
	"io/ioutil"
	"testing"

	"github.com/ActiveState/cli/internal/rtutils/singlethread"
	"github.com/stretchr/testify/assert"
)

func TestNewTracker(t *testing.T) {
	dir, err := ioutil.TempDir("", "")
	assert.NoError(t, err)

	tracker, err := newCustom(dir, singlethread.New(), true)
	assert.NoError(t, err)

	err = tracker.Close()
	assert.NoError(t, err)
}
