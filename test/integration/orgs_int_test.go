package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
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
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
}

func TestOrganizationsIntegrationTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(OrganizationsIntegrationTestSuite))
}
