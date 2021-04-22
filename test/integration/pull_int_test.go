package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type PullIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *PullIntegrationTestSuite) TestPull() {
	suite.OnlyRunForTags(tagsuite.Pull)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(`project: "https://platform.activestate.com/ActiveState-CLI/Pull?commitID=b490d8f1-9e0c-4294-a9a1-88c5957665b5"`)

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("activate"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.ExpectLongString("default project?")
	cp.Send("n")
	cp.WaitForInput(60 * time.Second)

	cp.SendLine(`python3 -c "import requests"`)
	cp.Expect("ModuleNotFoundError")

	cp = ts.Spawn("pull")
	cp.Expect("activestate.yaml has been updated")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("pull")
	cp.Expect("already up to date")
	cp.ExpectExitCode(0)

	cp = ts.SpawnCmd("python3", "-c", `"import requests"`)
	cp.ExpectExitCode(0)
}

func (suite *PullIntegrationTestSuite) TestPullSetProject() {
	suite.OnlyRunForTags(tagsuite.Pull)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(`project: https://platform.activestate.com/ActiveState-CLI/small-python?commitID=9733d11a-dfb3-41de-a37a-843b7c421db4`)

	// update to related project
	cp := ts.Spawn("pull", "--set-project", "ActiveState-CLI/small-python-fork")
	cp.Expect("activestate.yaml has been updated")
	cp.ExpectExitCode(0)
}

func (suite *PullIntegrationTestSuite) TestPullSetProjectUnrelated() {
	suite.OnlyRunForTags(tagsuite.Pull)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareActiveStateYAML(`project: "https://platform.activestate.com/ActiveState-CLI/small-python?commitID=9733d11a-dfb3-41de-a37a-843b7c421db4"`)

	cp := ts.Spawn("pull", "--set-project", "ActiveState-CLI/Python3")
	cp.ExpectLongString("you may lose changes to your project")
	cp.SendLine("n")
	cp.Expect("Pull aborted by user")
	cp.ExpectNotExitCode(0)

	cp = ts.Spawn("pull", "--force", "--set-project", "ActiveState-CLI/Python3")
	cp.Expect("activestate.yaml has been updated")
	cp.ExpectExitCode(0)
}

func TestPullIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PullIntegrationTestSuite))
}
