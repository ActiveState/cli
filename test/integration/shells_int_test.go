package integration

import (
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/ActiveState/termtest"
	"github.com/stretchr/testify/suite"
)

type ShellsIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ShellsIntegrationTestSuite) TestShells() {
	suite.OnlyRunForTags(tagsuite.Critical, tagsuite.Shell)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	var shells []string
	if runtime.GOOS == "linux" {
		shells = []string{"bash", "fish"}
	} else if runtime.GOOS == "darwin" {
		shells = []string{"bash", "zsh", "tcsh"}
	} else if runtime.GOOS == "windows" {
		shells = []string{"cmd"}
	}

	// Checkout the first instance. It doesn't matter which shell is used.
	cp := ts.SpawnWithOpts(e2e.WithArgs("checkout", "ActiveState-CLI/small-python"))
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	for _, shell := range shells {
		var cp *termtest.ConsoleProcess
		// Run the checkout in a particular shell.
		args := "checkout ActiveState-CLI/small-python " + shell
		if shell == "bash" {
			cp = ts.SpawnInBash(args)
		} else if shell == "zsh" {
			cp = ts.SpawnInZsh(args)
		} else if shell == "tcsh" {
			cp = ts.SpawnInTcsh(args)
		} else if shell == "fish" {
			cp = ts.SpawnInFish(args)
		} else if shell == "cmd" {
			cp = ts.SpawnInCmd(args)
		}
		cp.Expect("Checked out project")
		if shell != "cmd" {
			cp.ExpectExitCode(0)
		}

		// There are 2 or more instances checked out, so we should get a prompt in whichever shell we
		// use.
		args = "shell small-python"
		env := e2e.AppendEnv(
			"ACTIVESTATE_CLI_DISABLE_RUNTIME=false",
			"SHELL="+shell,
		)
		if shell == "bash" {
			cp = ts.SpawnInBash(args, env)
		} else if shell == "zsh" {
			cp = ts.SpawnInZsh(args, env)
		} else if shell == "tcsh" {
			cp = ts.SpawnInTcsh(args, env)
		} else if shell == "fish" {
			cp = ts.SpawnInFish(args, env)
		} else if shell == "cmd" {
			cp = ts.SpawnInCmd(args, env)
		}
		cp.Expect("Multiple project paths")
		cp.SendLine("\n")      // just pick the first one
		cp.Expect("Activated") // this means the selection prompt worked
		if shell != "tcsh" {   // tcsh prompt does not behave like other shells' prompts
			cp.Expect("[ActiveState-CLI/small-python]") // verify shell prompt contains the right info
		}
		cp.WaitForInput()
		cp.SendLine("python3 --version")
		cp.Expect("Python 3.10") // verify runtime is functioning properly
		if shell != "cmd" {
			cp.SendLine("echo $0")
			cp.Expect(shell) // verify the expected shell is running
		} else {
			cp.SendLine("echo %COMSPEC%")
			cp.Expect("cmd.exe") // verify the expected shell is running
		}
		cp.SendLine("exit")
		cp.Expect("Deactivated")
		if shell != "cmd" {
			cp.ExpectExitCode(0) // verify exiting the shell worked
		}
	}
}

func TestShellsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ShellsIntegrationTestSuite))
}
