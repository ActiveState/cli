package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type UseIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *UseIntegrationTestSuite) TestUse() {
	suite.OnlyRunForTags(tagsuite.Use)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(
		e2e.WithArgs("use", "ActiveState-CLI/Python3"),
		e2e.AppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Switched to Python3")

	cp.SendLine("hash -r")
	cp.WaitForInput()

	cp.SendLine("python3 --version")
	cp.Expect("Python 3.6.6")
}

func TestUseIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(UseIntegrationTestSuite))
}
