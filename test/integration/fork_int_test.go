package integration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/stretchr/testify/suite"
)

type ForkIntegrationTestSuite struct {
	suite.Suite
	username string
}

func (suite *ForkIntegrationTestSuite) cleanup(ts *e2e.Session) {
	cp := ts.Spawn("auth", "logout")
	cp.ExpectExitCode(0)
	ts.Close()
}

func (suite *ForkIntegrationTestSuite) TestFork_FailNameExists() {
	ts := e2e.New(suite.T(), false)
	defer suite.cleanup(ts)
	ts.LoginAsPersistentUser()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("fork", "ActiveState-CLI/Python3", "--org", e2e.PersistentUsername),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Could not create the forked project", 30*time.Second)
	cp.Expect("The name 'Python3' is no longer available, it was used in a now deleted project.", 30*time.Second)
	cp.ExpectNotExitCode(0)
	suite.NotContains(cp.TrimmedSnapshot(), "Successfully forked project")
}

func (suite *ForkIntegrationTestSuite) TestFork_EditorV0() {
	ts := e2e.New(suite.T(), false)
	defer suite.cleanup(ts)

	username := ts.CreateNewUser()

	results := struct {
		Result map[string]string `json:"result,omitempty"`
	}{
		map[string]string{
			"NewName":       "Test-Python3",
			"NewOwner":      username,
			"OriginalName":  "Python3",
			"OriginalOwner": "ActiveState-CLI",
		},
	}
	expected, err := json.Marshal(results)
	suite.Require().NoError(err)

	cp := ts.Spawn("fork", "ActiveState-CLI/Python3", "--name", "Test-Python3", "--org", username, "--output", "editor.v0")
	cp.Expect(`"OriginalOwner":"ActiveState-CLI"}}`)
	suite.Equal(string(expected), cp.TrimmedSnapshot())
	cp.ExpectExitCode(0)
}

func TestForkIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ForkIntegrationTestSuite))
}
