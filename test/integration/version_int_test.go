package integration

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/testsuite"
)

type VersionIntegrationTestSuite struct {
	testsuite.Suite
}

func (suite *VersionIntegrationTestSuite) TestNotDev() {
	suite.T().Log("If you aren't running this on CI you can safely ignore this test failing")

	suite.OnlyRunForTags(testsuite.TagCritical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("--version")
	suite.NotContains(cp.TrimmedSnapshot(), "(dev)")
	cp.ExpectExitCode(0)
}

func TestVersionIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(VersionIntegrationTestSuite))
}
