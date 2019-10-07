package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	"github.com/ActiveState/cli/internal/testhelpers/httpmock"
	"github.com/ActiveState/cli/internal/testhelpers/updatemocks"
	"github.com/ActiveState/cli/pkg/projectfile"
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
	updatemocks.MockUpdater(t, filepath.Join(testdatadir, "state.sh"), constants.BranchName, constants.Version)

	var args = []string{"somebinary", "arg1", "arg2", "--flag"}
	Command.Exiter = exiter.Exit
	exitCode := exiter.WaitForExit(func() {
		forwardAndExit(args)
	})
	assert.Equal(t, -1, exitCode, "exit was not called")
}

func TestForwardAndExit(t *testing.T) {
	httpmock.Activate(constants.APIUpdateURL)
	defer httpmock.DeActivate()
	Command.Exiter = exiter.Exit

	setupCwd(t, true)
	ext := ".sh"
	if runtime.GOOS == "windows" {
		ext = ".bat"
		forceFileExt = ".bat"
		defer func() { forceFileExt = "" }()
	}

	testdatadir := testdataDir(t)
	updatemocks.MockUpdater(t, filepath.Join(testdatadir, "state"+ext), "master", "1.2.3-123")

	var exitCode int
	var args = []string{"somebinary", "arg1", "arg2", "--flag"}
	out := capturer.CaptureStdout(func() {
		exitCode = exiter.WaitForExit(func() {
			forwardAndExit(args)
		})
	})
	require.Equal(t, 0, exitCode, "exits with code 0, output was:\n "+out)

	// Invoking the individual methods so we can capture stdout properly
	versionInfo := &projectfile.VersionInfo{Branch: "master", Version: "1.2.3-123"}
	binary := forwardBin(versionInfo)
	assert.Contains(t, binary, versionInfo.Branch, "Binary includes branch name")
	assert.Contains(t, binary, versionInfo.Version, "Binary includes version ")
	require.NotEmpty(t, binary, "Binary is set")

	out = capturer.CaptureStdout(func() {
		exitCode = exiter.WaitForExit(func() {
			execForwardAndExit(binary, args)
		})
	})
	require.Equal(t, 0, exitCode, "exits with code 0, output was:\n "+out)

	assert.Contains(t, out, fmt.Sprintf("OUTPUT--%s--OUTPUT", strings.Join(args[1:], " ")), "state.sh mock should print our args")
}
