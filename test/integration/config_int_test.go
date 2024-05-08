package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
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
	ts.IgnoreLogErrors()

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

func (suite *ConfigIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Config, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("config", "set", constants.UnstableConfig, "true", "-o", "json")
	cp.Expect(`"name":`)
	cp.Expect(`"value":`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)

	cp = ts.Spawn("config", "get", constants.UnstableConfig, "-o", "json")
	cp.Expect(`"name":`)
	cp.Expect(`"value":`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
}

func TestConfigIntegrationTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ConfigIntegrationTestSuite))
}
