package integration

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type ShellsIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ShellsIntegrationTestSuite) TestShells() {
	suite.OnlyRunForTags(tagsuite.Critical, tagsuite.Shell)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	var shells []e2e.Shell
	switch runtime.GOOS {
	case "linux":
		shells = []e2e.Shell{e2e.Bash, e2e.Fish, e2e.Tcsh, e2e.Zsh}
	case "darwin":
		shells = []e2e.Shell{e2e.Bash, e2e.Fish, e2e.Zsh, e2e.Tcsh}
	case "windows":
		shells = []e2e.Shell{e2e.Bash, e2e.Cmd}
	}

	// Checkout the first instance. It doesn't matter which shell is used.
	cp := ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/small-python"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	for _, shell := range shells {
		suite.T().Run(fmt.Sprintf("using_%s", shell), func(t *testing.T) {
			ts.SetT(t)

			if shell == e2e.Zsh {
				err := fileutils.Touch(filepath.Join(ts.Dirs.HomeDir, ".zshrc"))
				suite.Require().NoError(err)
			}

			// Run the checkout in a particular shell.
			cp = ts.SpawnShellWithOpts(shell)
			cp.SendLine(e2e.QuoteCommand(shell, ts.ExecutablePath(), "checkout", "ActiveState-CLI/small-python", string(shell)))
			cp.Expect("Checked out project")
			cp.SendLine("exit")
			if shell != e2e.Cmd {
				cp.ExpectExitCode(0)
			}

			// There are 2 or more instances checked out, so we should get a prompt in whichever shell we
			// use.
			cp = ts.SpawnShellWithOpts(shell, e2e.OptAppendEnv(constants.DisableRuntime+"=false"))
			cp.SendLine(e2e.QuoteCommand(shell, ts.ExecutablePath(), "shell", "small-python"))
			cp.Expect("Multiple project paths")

			// Just pick the first one and verify the selection prompt works.
			cp.SendEnter()
			cp.Expect("Activated", e2e.RuntimeSourcingTimeoutOpt)

			// Verify that the command prompt contains the right info, except for tcsh, whose prompt does
			// not behave like other shells'.
			if shell != e2e.Tcsh {
				cp.Expect("[ActiveState-CLI/small-python]")
			}

			// Verify the runtime is functioning properly.
			cp.SendLine("python3 --version")
			cp.Expect("Python 3.10")

			// Verify the expected shell is running.
			switch shell {
			case e2e.Cmd:
				cp.SendLine("echo %COMSPEC%")
				cp.Expect(string(shell))
			case e2e.Fish:
				cp.SendLine("echo $fish_pid")
				cp.ExpectRe("\\d+")
			default:
				cp.SendLine("echo $0")
				cp.Expect(string(shell))
			}

			// Verify exiting the shell works.
			cp.SendLine("exit")
			cp.Expect("Deactivated")

			// Exit the spawned shell.
			cp.SendLine("exit")
			cp.ExpectExitCode(0)
		})
	}
}

func TestShellsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ShellsIntegrationTestSuite))
}
