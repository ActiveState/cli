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

	projectsDir := filepath.Join(ts.Dirs.Base, "projects")

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("use", "ActiveState-CLI/Python3"),
		e2e.AppendEnv(
			"ACTIVESTATE_CLI_DISABLE_RUNTIME=false",
			"ACTIVESTATE_CLI_PROJECTSDIR="+projectsDir),
	)
	cp.Expect("Switched to Python3")

	args := []string{"Python3", "ActiveState-CLI/Python3"}
	for _, arg := range args {
		cp := ts.SpawnWithOpts(
			e2e.WithArgs("shell", arg),
			e2e.AppendEnv(
				"ACTIVESTATE_CLI_DISABLE_RUNTIME=false",
				"ACTIVESTATE_CLI_PROJECTSDIR="+projectsDir),
		)
		cp.Expect("Activated")
		cp.WaitForInput()

		cp.SendLine("python3 --version")
		cp.Expect("Python 3.6.6")
		cp.SendLine("exit")
		cp.ExpectExitCode(0)
	}

	// Check for project not checked out.
	args = []string{"Python-3.9", "ActiveState-CLI/Python-3.9"}
	for _, arg := range args {
		cp := ts.SpawnWithOpts(
			e2e.WithArgs("shell", arg),
			e2e.AppendEnv(
				"ACTIVESTATE_CLI_DISABLE_RUNTIME=false",
				"ACTIVESTATE_CLI_PROJECTSDIR="+projectsDir),
		)
		cp.Expect("The project Python-3.9 does not exist")
		cp.ExpectExitCode(1)
	}
}

func TestShellIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ShellIntegrationTestSuite))
}
