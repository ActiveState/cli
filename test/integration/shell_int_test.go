package integration

import (
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type ShellIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ShellIntegrationTestSuite) TestShell() {
	suite.OnlyRunForTags(tagsuite.Shell)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("get", "ActiveState-CLI/Python3"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Checked out Python3")
	cp.ExpectExitCode(0)

	args := []string{"Python3", "ActiveState-CLI/Python3"}
	for _, arg := range args {
		cp := ts.SpawnWithOpts(
			e2e.WithArgs("shell", arg),
			e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
		)
		cp.Expect("Activated")
		cp.WaitForInput()

		cp.SendLine("python3 --version")
		cp.Expect("Python 3.6.6")
		cp.SendLine("exit")
		cp.Expect("Deactivated")
		cp.ExpectExitCode(0)
	}

	// Check for project not checked out.
	args = []string{"Python-3.9", "ActiveState-CLI/Python-3.9"}
	for _, arg := range args {
		cp := ts.SpawnWithOpts(
			e2e.WithArgs("shell", arg),
			e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
		)
		cp.Expect("Cannot find the Python-3.9 project")
		cp.ExpectExitCode(1)
	}
}

func (suite *ShellIntegrationTestSuite) TestDefaultShell() {
	suite.OnlyRunForTags(tagsuite.Shell)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate", "ActiveState-CLI/Python3", "--default", "--path", filepath.Join(ts.Dirs.Work, "foo", "bar", "baz")),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Activated")
	cp.WaitForInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.WithArgs("shell"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Activated")
	cp.WaitForInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func (suite *ShellIntegrationTestSuite) TestCwdShell() {
	suite.OnlyRunForTags(tagsuite.Shell)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate", "ActiveState-CLI/Python3"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Activated")
	cp.WaitForInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.WithArgs("shell"),
		e2e.WithWorkDirectory(filepath.Join(ts.Dirs.Work, "Python3")),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Activated")
	cp.WaitForInput()
	cp.SendLine("exit")
	cp.ExpectExitCode(0)
}

func TestShellIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ShellIntegrationTestSuite))
}
