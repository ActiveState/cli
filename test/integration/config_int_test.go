package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal-as/testhelpers/e2e"
	"github.com/ActiveState/cli/internal-as/testhelpers/tagsuite"
	"github.com/ActiveState/cli/internal/constants"
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

	cp = ts.Spawn("config", "set", constants.UnstableConfig, "true")
	cp.Expect("Successfully")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("config", "get", constants.UnstableConfig)
	cp.Expect("true")

	cp = ts.Spawn("config", "set", constants.UnstableConfig, "false")
	cp.Expect("Successfully")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("config", "get", constants.UnstableConfig)
	cp.Expect("false")

	cp = ts.Spawn("config", "set", constants.UnstableConfig, "oops")
	cp.Expect("Invalid boolean value")
}

func TestConfigIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigIntegrationTestSuite))
}
