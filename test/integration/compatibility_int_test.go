package integration

import (
	"testing"

	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type CompatibilityIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *CompatibilityIntegrationTestSuite) TestOSVersionNotCompatible() {
	suite.OnlyRunForTags(tagsuite.Compatibility, tagsuite.Critical)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.SpawnWithOpts(e2e.WithArgs("--version"), e2e.AppendEnv(sysinfo.VersionOverrideEnvVar+"=10.0.0"))
	suite.NotContains(cp.TrimmedSnapshot(), "not compatible")
	cp.ExpectExitCode(1)
}

func TestCompatibilityIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(CompatibilityIntegrationTestSuite))
}
