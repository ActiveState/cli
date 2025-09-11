package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type HelloIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *HelloIntegrationTestSuite) TestHello() {
	suite.OnlyRunForTags(tagsuite.HelloExample)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()

	cp := ts.Spawn("_hello", "Person")
	cp.Expect("Hello, Person!")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("_hello", "")
	cp.Expect("Cannot say hello because no name was provided")
	cp.ExpectNotExitCode(0)
	ts.IgnoreLogErrors()

	cp = ts.Spawn("_hello", "Person", "--extra")
	cp.Expect("You are on commit")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("_hello", "Person", "--echo", "example")
	cp.Expect("Echoing: example")
	cp.ExpectExitCode(0)
}

func TestHelloIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(HelloIntegrationTestSuite))
}
