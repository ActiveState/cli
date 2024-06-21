package integration

import (
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type BundleIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *BundleIntegrationTestSuite) TestBundle_listingSimple() {
	suite.OnlyRunForTags(tagsuite.Bundle)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("bundles")
	cp.Expect("Name")
	cp.Expect("Desktop-Installer-Tools")
	cp.ExpectExitCode(0)
}

func (suite *BundleIntegrationTestSuite) TestBundle_project_name_noData() {
	suite.OnlyRunForTags(tagsuite.Bundle)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("bundles", "--namespace", "ActiveState/Perl-5.32", "--bundle", "Temp")
	cp.Expect("The project has no bundles to list.")
	cp.ExpectExitCode(0)
}

func (suite *BundleIntegrationTestSuite) TestBundle_searchSimple() {
	suite.OnlyRunForTags(tagsuite.Bundle)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	// Note that the expected strings might change due to inventory changes
	cp := ts.Spawn("bundles", "search", "Ut")
	expectations := []string{
		"Name",
		"Utilities",
		"1.00",
	}
	for _, expectation := range expectations {
		cp.Expect(expectation)
	}
	cp.Send("q")
	cp.ExpectExitCode(0)
}

func (suite *BundleIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session) {
	ts.PrepareProject("ActiveState-CLI-Testing/Perl-5.32", "3cbcdcba-df34-49ea-81d0-f1385603037d")
}

func (suite *BundleIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Bundle, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("bundles", "search", "Email", "--language", "Perl", "-o", "json")
	cp.Expect(`"Name":"Email"`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("checkout", "ActiveState-CLI/Bundles", "."),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("bundles", "install", "Testing", "--output", "json"),
	)
	cp.Expect(`"name":"Testing"`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("bundles", "uninstall", "Testing", "-o", "editor"),
	)
	cp.Expect(`"name":"Testing"`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
}

func TestBundleIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(BundleIntegrationTestSuite))
}
