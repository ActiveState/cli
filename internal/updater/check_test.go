package updater

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/config" // MUST be first!
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T, withVersion bool) {
	cwd, err := environment.GetRootPath()
	assert.NoError(t, err, "Should fetch cwd")
	path := filepath.Join(cwd, "internal", "updater", "testdata")
	if withVersion {
		path = filepath.Join(path, "withversion")
	}
	err = os.Chdir(path)
	assert.NoError(t, err, "Should change dir without issue.")
	projectfile.Reset()
}

func TestTimedCheck(t *testing.T) {
	setup(t, false)

	updateCheckMarker := filepath.Join(config.GetDataDir(), "update-check")
	os.Remove(updateCheckMarker) // remove if exists
	_, err := os.Stat(updateCheckMarker)
	assert.Error(t, err, "update-check marker does not exist")

	update := TimedCheck()
	assert.True(t, update, "Should want to update")

	stat, err := os.Stat(updateCheckMarker)
	assert.NoError(t, err, "update-check marker was created")
	modTime := stat.ModTime()

	update = TimedCheck()
	assert.False(t, update, "Should not want to update")
	stat, err = os.Stat(updateCheckMarker)
	assert.NoError(t, err, "update-check marker still exists")
	assert.Equal(t, modTime, stat.ModTime(), "update-check marker will not be modified for at least a day")
}

func TestTimedCheckLockedVersion(t *testing.T) {
	setup(t, true)

	updateCheckMarker := filepath.Join(config.GetDataDir(), "update-check")
	os.Remove(updateCheckMarker) // remove if exists
	_, err := os.Stat(updateCheckMarker)
	assert.Error(t, err, "update-check marker does not exist")

	update := TimedCheck()
	assert.False(t, update, "Should not want to update because we're using a locked version")
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
