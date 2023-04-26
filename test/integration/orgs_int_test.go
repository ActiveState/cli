package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type OrganizationsIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *OrganizationsIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Organizations, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()
	cp := ts.Spawn("organizations", "-o", "json")
	cp.Expect(`[{"name":`)
	cp.Expect(`}]`)
	AssertNoPlainOutput(suite.T(), cp)
	cp.ExpectExitCode(0)
}

func TestOrganizationsIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(OrganizationsIntegrationTestSuite))
}
