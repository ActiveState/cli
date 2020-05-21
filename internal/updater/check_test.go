package updater

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/config" // MUST be first!
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/updatemocks"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func setup(t *testing.T, withVersion bool) {
	cwd, err := environment.GetRootPath()
	require.NoError(t, err, "Should fetch cwd")
	path := filepath.Join(cwd, "internal", "updater", "testdata")
	if withVersion {
		path = filepath.Join(path, "withversion")
	}
	err = os.Chdir(path)
	require.NoError(t, err, "Should change dir without issue.")
	projectfile.Reset()
}

func TestTimedCheck(t *testing.T) {
	setup(t, false)

	updateCheckMarker := filepath.Join(config.ConfigPath(), "update-check")
	os.Remove(updateCheckMarker) // remove if exists
	_, err := os.Stat(updateCheckMarker)
	assert.Error(t, err, "update-check marker does not exist")

	httpmock.Activate(constants.APIUpdateURL)
	defer httpmock.DeActivate()

	updatemocks.MockUpdater(t, os.Args[0], constants.BranchName, "1.2.3-123")

	update, _ := AutoUpdate(configPath)
	assert.True(t, update, "Should want to update")

	stat, err := os.Stat(updateCheckMarker)
	assert.NoError(t, err, "update-check marker was created")
	modTime := stat.ModTime()

	update, _ = AutoUpdate(configPath)
	assert.False(t, update, "Should not want to update")
	stat, err = os.Stat(updateCheckMarker)
	assert.NoError(t, err, "update-check marker still exists")
	assert.Equal(t, modTime, stat.ModTime(), "update-check marker will not be modified for at least a day")
}

func TestTimedCheckLockedVersion(t *testing.T) {
	setup(t, true)

	updateCheckMarker := filepath.Join(config.ConfigPath(), "update-check")
	os.Remove(updateCheckMarker) // remove if exists
	_, err := os.Stat(updateCheckMarker)
	assert.Error(t, err, "update-check marker does not exist")

	update, _ := AutoUpdate(configPathWithVersion)
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
