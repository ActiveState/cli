package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/stretchr/testify/suite"
)

type HistoryIntegrationTestSuite struct {
	integration.Suite
	cleanup func()
}

func (suite *HistoryIntegrationTestSuite) TestHistory_History() {
	suite.LoginAsPersistentUser()
	suite.Spawn("history", "--namespace", "ActiveState-CLI/History")
	suite.Expect(`Platform  added`)
	suite.Wait()
}

func TestHistoryIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(HistoryIntegrationTestSuite))
}
