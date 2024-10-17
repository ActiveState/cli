package integration

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	vulnModel "github.com/ActiveState/cli/pkg/platform/api/vulnerabilities/model"
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

func (suite *ConfigIntegrationTestSuite) TestEnum() {
	suite.OnlyRunForTags(tagsuite.Config)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("config", "get", constants.SecurityPromptLevelConfig)
	cp.Expect(vulnModel.SeverityCritical)

	severities := []string{
		vulnModel.SeverityCritical,
		vulnModel.SeverityHigh,
		vulnModel.SeverityMedium,
		vulnModel.SeverityLow,
	}

	cp = ts.Spawn("config", "set", constants.SecurityPromptLevelConfig, "invalid")
	cp.Expect("Invalid value 'invalid': expected one of: " + strings.Join(severities, ", "))
	cp.ExpectNotExitCode(0)

	cp = ts.Spawn("config", "set", constants.SecurityPromptLevelConfig, vulnModel.SeverityLow)
	cp.ExpectExitCode(0)

	cp = ts.Spawn("config", "get", constants.SecurityPromptLevelConfig)
	cp.Expect(vulnModel.SeverityLow)
	cp.ExpectExitCode(0)
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

func (suite *ConfigIntegrationTestSuite) TestList() {
	suite.OnlyRunForTags(tagsuite.Config)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("config")
	cp.Expect("Key")
	cp.Expect("Value")
	cp.Expect("Default")
	cp.Expect("optin.buildscripts")
	cp.Expect("false")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("config", "set", "optin.buildscripts", "true")
	cp.Expect("Successfully")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("config")
	cp.Expect("Key")
	cp.Expect("Value")
	cp.Expect("Default")
	cp.Expect("optin.buildscripts")
	cp.Expect("true*")
	cp.ExpectExitCode(0)

	suite.Require().NotContains(cp.Snapshot(), constants.AsyncRuntimeConfig)
}
func TestConfigIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigIntegrationTestSuite))
}
