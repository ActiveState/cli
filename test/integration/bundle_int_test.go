package integration

import (
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

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
	cp.ExpectLongString("The requested project junk/junk could not be found.")
	cp.ExpectExitCode(1)
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
	cp.ExpectExitCode(0)
}

func (suite *BundleIntegrationTestSuite) TestBundle_searchWithExactTermWrongTerm() {
	suite.OnlyRunForTags(tagsuite.Bundle)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("bundles", "search", "xxxUtilitiesxxx", "--exact-term")
	cp.ExpectLongString("No bundles in our catalog match")
	cp.ExpectExitCode(1)
}

func (suite *BundleIntegrationTestSuite) TestBundle_searchWithLang() {
	suite.OnlyRunForTags(tagsuite.Bundle)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("bundles", "search", "Utilities", "--language=perl")
	cp.Expect("Utilities")
	cp.ExpectExitCode(0)
}

func (suite *BundleIntegrationTestSuite) TestBundle_searchWithWrongLang() {
	suite.OnlyRunForTags(tagsuite.Bundle)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("bundles", "search", "Utilities", "--language=python")
	cp.ExpectLongString("No bundles in our catalog match")
	cp.ExpectExitCode(1)
}

func (suite *BundleIntegrationTestSuite) TestBundle_searchWithBadLang() {
	suite.OnlyRunForTags(tagsuite.Bundle)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("bundles", "search", "Utilities", "--language=bad")
	cp.Expect("Cannot obtain search")
	cp.ExpectExitCode(1)
}

func (suite *BundleIntegrationTestSuite) TestBundle_headless_operation() {
	suite.OnlyRunForTags(tagsuite.Bundle)
	if runtime.GOOS == "darwin" {
		suite.T().Skip("Skipping mac for now as the builds are still too unreliable")
		return
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("activate", "ActiveState/Perl-5.32", "--path", ts.Dirs.Work, "--output=json")
	cp.ExpectExitCode(0)

	suite.Run("install non-existing", func() {
		cp := ts.Spawn("bundles", "install", "non-existing")
		cp.Expect("No results found for search term")
		cp.ExpectLongString(`Run "state search non-existing" to find alternatives`)
		cp.Wait()
	})

	suite.Run("install", func() {
		cp := ts.Spawn("bundles", "install", "Utilities")
		cp.ExpectRe("successfully installed", 45*time.Second)
		cp.Wait()
	})

	/* Our bundles have only one version currently.
	suite.Run("install (update)", func() {
		cp := ts.Spawn("bundles", "install", "Utilities@0.7.6")
		cp.ExpectRe("(?:bundle updated|being built)")
		cp.ExpectExitCode(1)
	})
	*/

	suite.Run("uninstall", func() {
		cp := ts.Spawn("bundles", "uninstall", "Utilities")
		cp.ExpectRe("(?:Bundle removed|being built)", 30*time.Second)
		cp.Wait()
	})
}

func (suite *BundleIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session) {
	asyData := `project: "https://platform.activestate.com/ActiveState/Perl-5.32?commitID=c9b1b41a-a153-46fb-b18d-3caa38e19377"`
	ts.PrepareActiveStateYAML(asyData)
}

func TestBundleIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(BundleIntegrationTestSuite))
}
