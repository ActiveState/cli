package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type DeleteUUIDProjectIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *DeleteUUIDProjectIntegrationTestSuite) TestRun() {
	suite.OnlyRunForTags(tagsuite.DeleteProjects)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	ts.DeleteUUIDProjects(e2e.PersistentUsername)
}

func TestDeleteUUIDProjectIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(DeleteUUIDProjectIntegrationTestSuite))
}
