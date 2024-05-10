package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type VersionIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *VersionIntegrationTestSuite) TestNotDev() {
	suite.T().Log("If you aren't running this on CI you can safely ignore this test failing")

	suite.OnlyRunForTags(tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("--version")
	suite.NotContains(cp.Output(), "(dev)")
	cp.ExpectExitCode(0)
}

func TestVersionIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(VersionIntegrationTestSuite))
}
