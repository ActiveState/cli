package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type ConfigIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ConfigIntegrationTestSuite) TestConfig() {
	suite.OnlyRunForTags(tagsuite.Config)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("config", "set", "invalid++", "value")
	cp.Expect("Invalid")
	cp.ExpectExitCode(1)

	cp = ts.Spawn("config", "set", "foo", "bar")
	cp.Expect("Successfully")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("config", "get", "foo")
	cp.Expect("bar")
}

func TestConfigIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigIntegrationTestSuite))
}
