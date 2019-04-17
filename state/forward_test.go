package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/updatemocks"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testdataDir(t *testing.T) string {
	cwd, err := environment.GetRootPath()
	require.NoError(t, err, "Should fetch cwd")
	return filepath.Join(cwd, "state", "testdata")
}

func setupCwd(t *testing.T, withVersion bool) {
	testdatadir := testdataDir(t)
	if withVersion {
		testdatadir = filepath.Join(testdatadir, "withversion")
	}
	err := os.Chdir(testdatadir)
	require.NoError(t, err, "Should change dir without issue.")
	projectfile.Reset()
}

func TestForwardNotUsed(t *testing.T) {
	httpmock.Activate(constants.APIUpdateURL)
	defer httpmock.DeActivate()

	setupCwd(t, false)
	testdatadir := testdataDir(t)
	updatemocks.MockUpdater(t, filepath.Join(testdatadir, "state.sh"), constants.Version)

	var args = []string{"somebinary", "arg1", "arg2", "--flag"}
	exit = exiter.Exit
	exitCode := exiter.WaitForExit(func() {
		forwardAndExit(args)
	})
	assert.Equal(t, -1, exitCode, "exit was not called")
}
