package updater

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ActiveState/ActiveState-CLI/internal/config" // MUST be first!
	"github.com/stretchr/testify/assert"
)

func TestTimedCheck(t *testing.T) {
	updateCheckMarker := filepath.Join(config.GetDataDir(), "update-check")
	os.Remove(updateCheckMarker) // remove if exists
	_, err := os.Stat(updateCheckMarker)
	assert.Error(t, err, "update-check marker does not exist")

	TimedCheck()
	assert.True(t, true, "no panic")

	stat, err := os.Stat(updateCheckMarker)
	assert.NoError(t, err, "update-check marker was created")
	modTime := stat.ModTime()

	TimedCheck()
	stat, err = os.Stat(updateCheckMarker)
	assert.NoError(t, err, "update-check marker still exists")
	assert.Equal(t, modTime, stat.ModTime(), "update-check marker will not be modified for at least a day")
}

func TestTimeout(t *testing.T) {
	info, err := timeout(func() (*Info, error) {
		return &Info{}, nil
	}, time.Second)
	assert.NoError(t, err, "no timeout")
	assert.NotNil(t, info, "no timeout")

	info, err = timeout(func() (*Info, error) {
		return nil, errors.New("some error")
	}, time.Second)
	assert.Nil(t, info, "non-timeout error")
	assert.Equal(t, err.Error(), "some error", "non-timeout error")

	info, err = timeout(func() (*Info, error) {
		time.Sleep(time.Second)
		return &Info{}, nil
	}, time.Millisecond)
	assert.Nil(t, info, "timeout")
	assert.Equal(t, err.Error(), "timeout", "timeout error")
}
