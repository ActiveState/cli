package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type ManifestIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ManifestIntegrationTestSuite) TestManifest() {
	suite.OnlyRunForTags(tagsuite.Manifest)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState/cli#9eee7512-b2ab-4600-b78b-ab0cf2e817d8", "."),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("manifest"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Operating on project: ActiveState/cli", e2e.RuntimeSourcingTimeoutOpt)
	cp.Expect("Name")
	cp.Expect("python")
	cp.Expect("3.9.13")
	cp.Expect("1 Critical,")
	cp.Expect("psutil")
	cp.Expect("auto → 5.9.0")
	cp.Expect("None detected")
	cp.ExpectExitCode(0)
}

func (suite *ManifestIntegrationTestSuite) TestManifest_JSON() {
	suite.OnlyRunForTags(tagsuite.Manifest)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	cp := ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState/cli#9eee7512-b2ab-4600-b78b-ab0cf2e817d8", "."),
	)
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("manifest", "--output", "json"),
	)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
	cp.Expect(`"requirements":`)
}

func TestManifestIntegrationTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ManifestIntegrationTestSuite))
}
