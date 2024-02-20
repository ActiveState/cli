package integration

import (
	"runtime"
	"testing"
	"time"

	"github.com/ActiveState/termtest"
	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
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

func (suite *BundleIntegrationTestSuite) TestBundle_project() {
	suite.OnlyRunForTags(tagsuite.Bundle)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("bundles", "--namespace", "ActiveState/Perl-5.32")
	cp.Expect("Name")
	cp.Expect("Desktop-Installer-Tools")
	cp.ExpectExitCode(0)
}

func (suite *BundleIntegrationTestSuite) TestBundle_name() {
	suite.OnlyRunForTags(tagsuite.Bundle)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("bundles", "--bundle", "Desktop")
	cp.Expect("Name")
	cp.Expect("Desktop-Installer-Tools")
	cp.ExpectExitCode(0)
}

func (suite *BundleIntegrationTestSuite) TestBundle_project_name() {
	suite.OnlyRunForTags(tagsuite.Bundle)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("bundles", "--namespace", "ActiveState/Perl-5.32", "--bundle", "Desktop")
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

func (suite *BundleIntegrationTestSuite) TestBundle_project_invalid() {
	suite.OnlyRunForTags(tagsuite.Bundle)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("bundles", "--namespace", "junk/junk")
	cp.Expect("The requested project junk does not exist under junk")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
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

func (suite *BundleIntegrationTestSuite) TestBundle_searchWithExactTerm() {
	suite.OnlyRunForTags(tagsuite.Bundle)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("bundles", "search", "Utilities", "--exact-term")
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

func (suite *BundleIntegrationTestSuite) TestBundle_searchWithExactTermWrongTerm() {
	suite.OnlyRunForTags(tagsuite.Bundle)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("bundles", "search", "xxxUtilitiesxxx", "--exact-term")
	cp.Expect("No bundles in our catalog match")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *BundleIntegrationTestSuite) TestBundle_searchWithLang() {
	suite.OnlyRunForTags(tagsuite.Bundle)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("bundles", "search", "Utilities", "--language=perl")
	cp.Expect("Utilities")
	cp.Send("q")
	cp.ExpectExitCode(0)
}

func (suite *BundleIntegrationTestSuite) TestBundle_searchWithWrongLang() {
	suite.OnlyRunForTags(tagsuite.Bundle)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("bundles", "search", "Utilities", "--language=python")
	cp.Expect("No bundles in our catalog match")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *BundleIntegrationTestSuite) TestBundle_searchWithBadLang() {
	suite.OnlyRunForTags(tagsuite.Bundle)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("bundles", "search", "Utilities", "--language=bad")
	cp.Expect("Cannot obtain search")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *BundleIntegrationTestSuite) TestBundle_detached_operation() {
	suite.OnlyRunForTags(tagsuite.Bundle)
	if runtime.GOOS == "darwin" {
		suite.T().Skip("Skipping mac for now as the builds are still too unreliable")
		return
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState/Perl-5.32", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	suite.Run("install non-existing", func() {
		cp := ts.Spawn("bundles", "install", "non-existing")
		cp.Expect("No results found for search term")
		cp.Expect(`Run "state search non-existing" to find alternatives`)
		cp.Wait()
	})

	suite.Run("install", func() {
		cp := ts.Spawn("bundles", "install", "Utilities")
		cp.ExpectRe("successfully installed", termtest.OptExpectTimeout(45*time.Second))
		cp.Wait()
	})

	/* Our bundles have only one version currently.
	suite.Run("install (update)", func() {
		cp := ts.Spawn("bundles", "install", "Utilities@0.7.6")
		cp.ExpectRe("(?:bundle updated|being built)")
		cp.ExpectExitCode(1)
		ts.IgnoreLogErrors()
	})
	*/

	suite.Run("uninstall", func() {
		cp := ts.Spawn("bundles", "uninstall", "Utilities")
		cp.ExpectRe("Bundle uninstalled", termtest.OptExpectTimeout(30*time.Second))
		cp.Wait()
	})
}

func (suite *BundleIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session) {
	ts.PrepareProject("ActiveState/Perl-5.32", "c9b1b41a-a153-46fb-b18d-3caa38e19377")
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
		e2e.OptAppendEnv("ACTIVESTATE_CLI_DISABLE_RUNTIME=false"),
	)
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("bundles", "install", "Testing", "--output", "json"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect(`"name":"Testing"`, e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)

	cp = ts.SpawnWithOpts(
		e2e.OptArgs("bundles", "uninstall", "Testing", "-o", "editor"),
		e2e.OptAppendEnv(constants.DisableRuntime+"=false"),
	)
	cp.Expect(`"name":"Testing"`, e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
}

func TestBundleIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(BundleIntegrationTestSuite))
}
