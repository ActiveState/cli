package main

import (
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	depMock "github.com/ActiveState/cli/internal/deprecation/mock"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/testhelpers/exiter"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/stretchr/testify/require"

	"github.com/spf13/pflag"
	funk "github.com/thoas/go-funk"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	setupCwd(t, false)
	assert := assert.New(t)

	Cc := Command.GetCobraCmd()
	pflags := Cc.PersistentFlags()
	var flags []string

	pflags.VisitAll(func(pf *pflag.Flag) {
		flags = append(flags, pf.Name)
	})

	assert.Equal(funk.Contains(flags, "locale"), true, "locale pflag is set")
}

func TestMainFn(t *testing.T) {
	setupCwd(t, false)
	assert := assert.New(t)

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--help"})

	main()

	assert.Equal(true, true, "main didn't panic")
}

func TestMainFnVerbose(t *testing.T) {
	setupCwd(t, false)
	assert := assert.New(t)

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--verbose"})

	Flags.Verbose = true
	defer func() {
		Flags.Verbose = false
	}()
	onVerboseFlag()
	out, err := osutil.CaptureStderr(func() {
		logging.Debug("AM I VERBOSE?")
	})
	require.NoError(t, err)

	assert.Equal(true, true, "main didn't panic")
	assert.Contains(out, "AM I VERBOSE?")
}

func TestMainError(t *testing.T) {
	setupCwd(t, false)
	assert := assert.New(t)

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{"--foo"})

	exit = exiter.Exit
	exitCode := exiter.WaitForExit(main)

	assert.Equal(exitCode, 1, "main didn't exit")
}

func TestExecute(t *testing.T) {
	setupCwd(t, false)
	assert := assert.New(t)

	Cc := Command.GetCobraCmd()
	Cc.SetArgs([]string{})

	Execute(Cc, []string{"--help"})

	assert.Equal(true, true, "Execute didn't panic")
}

func TestUnstableWarning(t *testing.T) {
	setupCwd(t, false)
	defer func() { branchName = constants.BranchName }()
	branchName = "anything-but-stable"
	out, err := osutil.CaptureStderr(main)
	require.NoError(t, err)

	assert.Contains(t, out, locale.Tr("unstable_version_warning", constants.BugTrackerURL), "Prints our unstable warning")
}

func TestDeprecated(t *testing.T) {
	setupCwd(t, false)
	mock := depMock.Init()
	defer mock.Close()
	mock.MockDeprecated()

	out, err := osutil.CaptureStdout(main)
	require.NoError(t, err)
	require.Contains(t, out, locale.Tr("warn_deprecation", "")[0:50])
}

func TestExpired(t *testing.T) {
	setupCwd(t, false)
	mock := depMock.Init()
	defer mock.Close()
	mock.MockExpired()

	out, err := osutil.CaptureStderr(main)
	require.NoError(t, err)
	require.Contains(t, out, locale.Tr("err_deprecation", "")[0:50])
}
