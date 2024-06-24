package integration

import (
	"testing"

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

	cp := ts.Spawn("checkout", "ActiveState/cli#9eee7512-b2ab-4600-b78b-ab0cf2e817d8", ".")
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)

	cp = ts.Spawn("manifest")
	cp.Expect("Operating on project: ActiveState/cli")
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

	cp := ts.Spawn("checkout", "ActiveState/cli#9eee7512-b2ab-4600-b78b-ab0cf2e817d8", ".")
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)

	cp = ts.Spawn("manifest", "--output", "json")
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
	cp.Expect(`"requirements":`)
}

func TestManifestIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ManifestIntegrationTestSuite))
}
