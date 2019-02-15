package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	"github.com/ActiveState/cli/internal/testhelpers/updatemocks"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/stretchr/testify/assert"
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

func TestForwardAndExit(t *testing.T) {
	httpmock.Activate(constants.APIUpdateURL)
	defer httpmock.DeActivate()
	exit = exiter.Exit

	setupCwd(t, true)

	testdatadir := testdataDir(t)
	updatemocks.MockUpdater(t, filepath.Join(testdatadir, "state.sh"), "1.2.3-123")

	var args = []string{"somebinary", "arg1", "arg2", "--flag"}
	exitCode := exiter.WaitForExit(func() {
		forwardAndExit(args)
	})
	require.Equal(t, 0, exitCode, "exits with code 0")

	// Invoking the individual methods so we can capture stdout properly
	binary := forwardBin("1.2.3-123")
	out, err := osutil.CaptureStdout(func() {
		exitCode = exiter.WaitForExit(func() {
			execForwardAndExit(binary, args)
		})
	})
	require.Equal(t, 0, exitCode, "exits with code 0")
	require.NoError(t, err)

	assert.Contains(t, out, fmt.Sprintf("OUTPUT--%s--OUTPUT", strings.Join(args[1:], " ")), "state.sh mock should print our args")
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
