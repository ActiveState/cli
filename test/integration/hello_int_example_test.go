package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type HelloIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *HelloIntegrationTestSuite) TestHello() {
	suite.OnlyRunForTags(tagsuite.HelloExample)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/small-python", ".")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("hello")
	cp.Expect("Hello, Friend!")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("hello", "Person")
	cp.Expect("Hello, Person!")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("hello", "")
	cp.Expect("Cannot say hello")
	cp.Expect("No name provided")
	cp.ExpectNotExitCode(0)

	cp = ts.Spawn("hello", "--extra")
	cp.Expect("Project: ActiveState-CLI/small-python")
	cp.Expect("Current commit message:")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("hello", "--echo", "example")
	cp.Expect("Echoing: example")
	cp.ExpectExitCode(0)
}

func TestHelloIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(HelloIntegrationTestSuite))
}