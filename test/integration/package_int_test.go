package integration

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type PackageIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *PackageIntegrationTestSuite) TestPackage_listingSimple() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("packages")
	cp.Expect("Operating on project")
	cp.Expect("ActiveState-CLI/List")
	cp.Expect("Name")
	cp.Expect("pytest")
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestPackages_project() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("packages", "--namespace", "ActiveState-CLI/List")
	cp.Expect("Name")
	cp.Expect("numpy")
	cp.Expect("pytest")
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestPackages_name() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("packages", "--package", "py")
	cp.Expect("Name")
	cp.Expect("pytest")
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestPackages_project_name() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("packages", "--namespace", "ActiveState-CLI/List", "--package", "py")
	cp.Expect("Name")
	cp.Expect("pytest")
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestPackages_project_name_noData() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("packages", "--namespace", "ActiveState-CLI/List", "--package", "req")
	cp.Expect("The project has no packages to list.")
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestPackages_project_invalid() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("packages", "--namespace", "junk/junk")
	cp.Expect("The requested project junk does not exist under junk")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *PackageIntegrationTestSuite) TestPackage_listingWithCommitValid() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("packages", "--commit", "b350c879-b72a-48da-bbc2-d8d709a6182a")
	cp.Expect("Name")
	cp.Expect("numpy")
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestPackage_listingWithCommitInvalid() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("packages", "--commit", "junk")
	cp.Expect("Cannot obtain")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *PackageIntegrationTestSuite) TestPackage_listingWithCommitUnknown() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("packages", "--commit", "00010001-0001-0001-0001-000100010001")
	cp.Expect("No data")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *PackageIntegrationTestSuite) TestPackage_listingWithCommitValidNoPackages() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("packages", "--commit", "cd674adb-e89a-48ff-95c6-ad52a177537b")
	cp.Expect("The project has no packages to list.")
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchSimple() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	// Note that the expected strings might change due to inventory changes
	cp := ts.Spawn("search", "requests")
	expectations := []string{
		"requests2",
		"2.16.0",
	}
	for _, expectation := range expectations {
		cp.Expect(expectation)
	}
	cp.Send("q")
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchWithExactTerm() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("search", "requests", "--exact-term")
	expectations := []string{
		"Name",
		"requests",
		"more",
	}
	for _, expectation := range expectations {
		cp.Expect(expectation)
	}
	cp.Send("q")
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchWithExactTermWrongTerm() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("search", "Requests", "--exact-term")
	cp.Expect("No packages in our catalog match")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	cp = ts.Spawn("search", "xxxrequestsxxx", "--exact-term")
	cp.Expect("No packages in our catalog match")
	cp.ExpectExitCode(1)
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchWithLang() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("search", "Moose", "--language=perl")
	cp.Expect("Name")
	cp.Expect("Moose")
	cp.Expect("Moose-Autobox")
	cp.Expect("MooseFS")
	cp.Send("q")
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchModules() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("search", "leapsecond", "--language=perl")
	cp.Expect("Date-Leapsecond")
	cp.Expect("DateTime-LeapSecond")
	cp.Expect("DateTime-Lite")
	cp.Send("q")
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchWithWrongLang() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("search", "xxxjunkxxx", "--language=perl")
	cp.Expect("No packages in our catalog match")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchWithBadLang() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("search", "numpy", "--language=bad")
	cp.Expect("Cannot obtain search")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *PackageIntegrationTestSuite) TestPackage_info() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("info", "pexpect")
	cp.Expect("Package Information")
	cp.Expect("Authors")
	cp.Expect("Version")
	cp.Expect("Available")
	cp.Expect("What's next?")
	cp.Expect("run 'state install")
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestPackage_operation_multiple() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	suite.Run("install", func() {
		cp := ts.Spawn("install", "requests", "urllib3@1.25.6")
		cp.Expect("Operating on project ActiveState-CLI/small-python")
		cp.Expect("Added: language/python/requests", e2e.RuntimeSourcingTimeoutOpt)
		cp.Expect("Added: language/python/urllib3")
		cp.Wait()
	})

	suite.Run("install (update)", func() {
		cp := ts.Spawn("install", "urllib3@1.25.8")
		cp.Expect("Operating on project ActiveState-CLI/small-python")
		cp.Expect("Updated: language/python/urllib3", e2e.RuntimeSourcingTimeoutOpt)
		cp.Wait()
	})

	suite.Run("uninstall", func() {
		cp := ts.Spawn("uninstall", "requests", "urllib3")
		cp.Expect("Operating on project ActiveState-CLI/small-python")
		cp.Expect("Removed: language/python/requests", e2e.RuntimeSourcingTimeoutOpt)
		cp.Expect("Removed: language/python/urllib3")
		cp.Wait()
	})
}

func (suite *PackageIntegrationTestSuite) TestPackage_Duplicate() {
	suite.OnlyRunForTags(tagsuite.Package)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareEmptyProject()

	cp := ts.Spawn("install", "shared/zlib") // install
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "shared/zlib") // install again
	cp.Expect(" no changes")
	cp.ExpectNotExitCode(0)
	ts.IgnoreLogErrors()

	if strings.Count(cp.Snapshot(), " x ") != 2 { // 2 because "Creating commit x Failed" is also printed
		suite.Fail("Expected exactly ONE error message, got: ", cp.Snapshot())
	}
}

func (suite *PackageIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session) {
	asyData := `project: "https://platform.activestate.com/ActiveState-CLI/List"
scripts:
  - name: test-pyparsing
    language: python3
    value: |
      from pyparsing import Word, alphas
      print(Word(alphas).parseString("TEST"))
`
	ts.PrepareActiveStateYAML(asyData)
	ts.PrepareCommitIdFile("a9d0bc88-585a-49cf-89c1-6c07af781cff")
}

func (suite *PackageIntegrationTestSuite) TestPackage_Install() {
	suite.OnlyRunForTags(tagsuite.Package)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")
	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "requests")
	cp.Expect("project has been updated")
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestPackage_Uninstall() {
	suite.OnlyRunForTags(tagsuite.Package)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI-Testing/small-python-with-pkg", "a2115792-2620-4217-89ed-b596c8c11ce3")
	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("uninstall", "requests")
	cp.Expect("project has been updated")
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestPackage_UninstallDoesNotExist() {
	suite.OnlyRunForTags(tagsuite.Package)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("uninstall", "doesNotExist")
	cp.Expect("could not be found")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	if strings.Count(cp.Snapshot(), " x ") != 2 { // 2 because "Creating commit x Failed" is also printed
		suite.Fail("Expected exactly ONE error message, got: ", cp.Snapshot())
	}
}

func (suite *PackageIntegrationTestSuite) TestPackage_UninstallDupeMatch() {
	suite.OnlyRunForTags(tagsuite.Package)

	ts := e2e.New(suite.T(), true)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI-Testing/duplicate-pkg-name", "e5a15d59-9192-446a-a133-9f4c2ebe0898")
	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("uninstall", "oauth")
	cp.Expect("match multiple requirements")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()

	if strings.Count(cp.Snapshot(), " x ") != 2 { // 2 because "Creating commit x Failed" is also printed
		suite.Fail("Expected exactly ONE error message, got: ", cp.Snapshot())
	}

	cp = ts.Spawn("uninstall", "language/python/oauth")
	cp.Expect("project has been updated")
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestJSON() {
	suite.OnlyRunForTags(tagsuite.Package, tagsuite.JSON)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("search", "Text-CSV", "--exact-term", "--language", "Perl", "-o", "json")
	cp.Expect(`"Name":"Text-CSV"`)
	cp.ExpectExitCode(0)
	// AssertValidJSON(suite.T(), cp) // currently too large to fit terminal window to validate

	ts.PrepareProject("ActiveState-CLI/Packages-Perl", "b2feab96-f700-47a3-85ef-2ec44c390c6b")

	cp = ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "Text-CSV", "-o", "json")
	cp.Expect(`{"name":"Text-CSV"`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)

	cp = ts.Spawn("packages", "-o", "json")
	cp.Expect(`[{"package":"Text-CSV","version":"Auto","resolved_version":"`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)

	cp = ts.Spawn("uninstall", "Text-CSV", "-o", "json")
	cp.Expect(`{"name":"Text-CSV"`)
	cp.ExpectExitCode(0)
	AssertValidJSON(suite.T(), cp)
}

func (suite *PackageIntegrationTestSuite) TestInstall_InvalidVersion() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "pytest@999.9999.9999")
	// User facing error from build planner
	// We only assert the state tool curated part of the error as the underlying build planner error may change
	cp.Expect("Could not plan build")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *PackageIntegrationTestSuite) TestUpdate_InvalidVersion() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "pytest") // install
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "pytest@999.9999.9999") // update
	// User facing error from build planner
	// We only assert the state tool curated part of the error as the underlying build planner error may change
	cp.Expect("Could not plan build")
	cp.ExpectExitCode(1)
	ts.IgnoreLogErrors()
}

func (suite *PackageIntegrationTestSuite) TestUpdate() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "pytest@7.3.2") // install
	cp.ExpectExitCode(0)

	cp = ts.Spawn("history")
	cp.Expect("pytest")
	cp.Expect("7.3.2")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("packages")
	cp.Expect("pytest")
	cp.Expect("7.3.2")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "pytest@7.4.0") // update
	cp.ExpectExitCode(0)

	cp = ts.Spawn("history")
	cp.Expect("pytest")
	cp.Expect("7.4.0")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("packages")
	cp.Expect("pytest")
	cp.Expect("7.4.0")
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestRuby() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI-Testing/Ruby", "72fadc10-ed8c-4be6-810b-b3de6e017c57")

	cp := ts.Spawn("install", "rake")
	cp.ExpectExitCode(0, e2e.RuntimeSourcingTimeoutOpt)

	cp = ts.Spawn("exec", "rake", "--", "--version")
	cp.ExpectRe(`rake, version \d+\.\d+\.\d+`)
	cp.ExpectExitCode(0)
}

// TestProjectWithOfflineInstallerAndDocker just makes sure we can checkout and install/uninstall
// packages for projects with offline installers and docker runtimes.
func (suite *PackageIntegrationTestSuite) TestProjectWithOfflineInstallerAndDocker() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser() // needed for Enterprise-tier features

	cp := ts.Spawn("checkout", "ActiveState-CLI/Python-OfflineInstaller-Docker", ".")
	cp.Expect("Checked out project", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestResolved() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "requests")
	cp.ExpectExitCode(0, e2e.RuntimeSourcingTimeoutOpt)

	cp = ts.Spawn("packages")
	cp.Expect("Auto →")
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestCVE_NoPrompt() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "urllib3@2.0.2")
	cp.ExpectRe(`Warning: Dependency has \d+ known vulnerabilities`, e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestCVE_Prompt() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	ts.PrepareProject("ActiveState-CLi/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("config", "set", "security.prompt.level", "high")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("config", "set", constants.SecurityPromptConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "urllib3@2.0.2", "--ts=2024-09-10T16:36:34.393Z")
	cp.ExpectRe(`Warning: Dependency has .* vulnerabilities`, e2e.RuntimeSourcingTimeoutOpt)
	cp.Expect("Do you want to continue")
	cp.SendLine("y")
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestCVE_Indirect() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser()

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("config", "set", constants.SecurityPromptConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "private/ActiveState-CLI-Testing/language/python/django_dep", "--ts=2024-09-10T16:36:34.393Z")
	cp.ExpectRe(`Warning: Dependency has \d+ indirect known vulnerabilities`, e2e.RuntimeSourcingTimeoutOpt)
	cp.Expect("Do you want to continue")
	cp.SendLine("n")
	cp.ExpectExitCode(1)
}

func (suite *PackageIntegrationTestSuite) TestChangeSummary() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.Expect("Successfully set")
	cp.ExpectExitCode(0)

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")

	cp = ts.Spawn("install", "requests@2.31.0")
	cp.Expect("Resolving Dependencies")
	cp.Expect("Done")
	cp.Expect("Installing requests@2.31.0 includes 4 direct dependencies")
	cp.Expect("├─ ")
	cp.Expect("├─ ")
	cp.Expect("├─ ")
	cp.Expect("└─ ")
	cp.Expect("Added: language/python/requests", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestChangeSummaryShowsAddedForUpdate() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.Expect("Successfully set")
	cp.ExpectExitCode(0)

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")

	cp = ts.Spawn("install", "jinja2@2.0")
	cp.Expect("Added: language/python/jinja2")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("install", "jinja2@3.1.4")
	cp.Expect("Installing jinja2@3.1.4 includes 1 direct dep")
	cp.Expect("└─ markupsafe@2.1.5")
	cp.Expect("Updated: language/python/jinja2")
	cp.ExpectExitCode(0)
}

func TestPackageIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PackageIntegrationTestSuite))
}
