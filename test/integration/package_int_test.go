package integration

import (
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
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
	cp.ExpectLongString("The requested project junk/junk could not be found.")
	cp.ExpectExitCode(1)
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
}

func (suite *PackageIntegrationTestSuite) TestPackage_listingWithCommitUnknown() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("packages", "--commit", "00010001-0001-0001-0001-000100010001")
	cp.Expect("No data")
	cp.ExpectExitCode(1)
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
		"requests3",
		"3.0.0a1",
	}
	for _, expectation := range expectations {
		cp.Expect(expectation)
	}
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
		"2.25.0",
		"+ 8 older versions",
	}
	for _, expectation := range expectations {
		cp.ExpectLongString(expectation)
	}
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchWithExactTermWrongTerm() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("search", "xxxrequestsxxx", "--exact-term")
	cp.ExpectLongString("No packages in our catalogue match")
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
	cp.Expect("MooseFS")
	cp.Expect("MooseX-ABC")
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchModules() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("search", "leapsecond", "--language=perl")
	cp.Expect("Matching modules")
	cp.Expect("Date::Leapsecond")
	cp.Expect("Matching modules")
	cp.Expect("DateTime::LeapSecond")
	cp.Expect("Matching modules")
	cp.Expect("DateTime::LeapSecond")
	cp.Expect("Matching modules")
	cp.Expect("DateTime::Lite::LeapSecond")
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchWithWrongLang() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("search", "numpy", "--language=perl")
	cp.ExpectLongString("No packages in our catalogue match")
	cp.ExpectExitCode(1)
}

func (suite *PackageIntegrationTestSuite) TestPackage_searchWithBadLang() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("search", "numpy", "--language=bad")
	cp.Expect("Cannot obtain search")
	cp.ExpectExitCode(1)
}

func (suite *PackageIntegrationTestSuite) TestPackage_info() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()
	suite.PrepareActiveStateYAML(ts)

	cp := ts.Spawn("info", "pexpect")
	cp.Expect("Details for version")
	cp.Expect("Authors")
	cp.Expect(" - ")
	cp.Expect(" - ")
	cp.Expect("License")
	cp.Expect("GPL")
	cp.Expect("Version")
	cp.Expect("Available")
	cp.Expect("4.8.0")
	cp.Expect("4.7.0")
	cp.Expect("4.6.0")
	cp.Expect("What's next?")
	cp.Expect("run `state install")
	cp.ExpectExitCode(0)
}

const (
	reqsFileName = "requirements.txt"
	reqsData     = `Click==7.0
Flask==1.1.1
Flask-Cors==3.0.8
itsdangerous==1.1.0
Jinja2==2.10.3
MarkupSafe==1.1.1
packaging==20.1
pyparsing==2.4.6
six==1.14.0
Werkzeug==0.16.0
`
	badReqsData = `Click==7.0
garbage---<<001.X
six==1.14.0
`
)

func (suite *PackageIntegrationTestSuite) TestPackage_import() {
	suite.OnlyRunForTags(tagsuite.Package)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	username := ts.CreateNewUser()
	namespace := fmt.Sprintf("%s/%s", username, "Python3")

	cp := ts.Spawn("init", namespace, "python3", "--path="+ts.Dirs.Work, "--skeleton=editor")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("push")
	cp.Expect(fmt.Sprintf("Creating project Python3 under %s", username))
	cp.ExpectExitCode(0)

	reqsFilePath := filepath.Join(cp.WorkDirectory(), reqsFileName)

	suite.Run("uninstalled import fails", func() {
		cp := ts.Spawn("run", "test-pyparsing")
		cp.ExpectNotExitCode(0, time.Second*60)
	})

	suite.Run("invalid requirements.txt", func() {
		ts.PrepareFile(reqsFilePath, badReqsData)

		cp := ts.Spawn("import", "requirements.txt")
		cp.ExpectNotExitCode(0, time.Second*60)
	})

	suite.Run("valid requirements.txt", func() {
		ts.PrepareFile(reqsFilePath, reqsData)

		cp := ts.Spawn("import", "requirements.txt")
		cp.Expect("state pull")
		cp.ExpectExitCode(0, time.Second*60)

		suite.Run("uninstalled import fails", func() {
			cp := ts.Spawn("run", "test-pyparsing")
			cp.ExpectExitCode(0, time.Second*60)
		})

		suite.Run("already added", func() {
			cp := ts.Spawn("import", "requirements.txt")
			cp.Expect("Are you sure you want to do this")
			cp.Send("n")
			cp.ExpectNotExitCode(0, time.Second*60)
		})
	})
}

func (suite *PackageIntegrationTestSuite) TestPackage_headless_operation() {
	suite.OnlyRunForTags(tagsuite.Package)
	if runtime.GOOS == "darwin" {
		suite.T().Skip("Skipping mac for now as the builds are still too unreliable")
		return
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("activate", "ActiveState-CLI/small-python", "--path", ts.Dirs.Work, "--output=json")
	cp.ExpectExitCode(0)

	suite.Run("install non-existing", func() {
		cp := ts.Spawn("install", "json")
		cp.ExpectLongString("Do you want to continue as an anonymous user?")
		cp.Send("Y")
		cp.Expect("Could not match json")
		cp.Expect("json2")
		cp.ExpectLongString("to see more results run `state search json`")
		cp.Wait()
	})

	suite.Run("install", func() {
		cp := ts.Spawn("install", "dateparser@0.7.2")
		cp.ExpectLongString("Do you want to continue as an anonymous user?")
		cp.Send("Y")
		cp.ExpectRe("(?:Package added|project is currently building)", 30*time.Second)
		cp.Wait()
	})

	suite.Run("install (update)", func() {
		cp := ts.Spawn("install", "dateparser@0.7.6")
		cp.ExpectLongString("Any changes you make are local only")
		cp.ExpectRe("(?:Package updated|project is currently building)", 50*time.Second)
		cp.Wait()
	})

	suite.Run("uninstall", func() {
		cp := ts.Spawn("uninstall", "dateparser")
		cp.ExpectRe("(?:Package uninstalled|project is currently building)", 30*time.Second)
		cp.Wait()
	})
}

func (suite *PackageIntegrationTestSuite) TestPackage_operation() {
	suite.OnlyRunForTags(tagsuite.Package, tagsuite.Revert)
	if runtime.GOOS == "darwin" {
		suite.T().Skip("Skipping mac for now as the builds are still too unreliable")
		return
	}
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	username := ts.CreateNewUser()
	namespace := fmt.Sprintf("%s/%s", username, "python3-pkgtest")

	cp := ts.Spawn("fork", "ActiveState-CLI/Revert", "--org", username, "--name", "python3-pkgtest")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("activate", namespace, "--path="+ts.Dirs.Work, "--output=json")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("history", "--output=json")
	cp.ExpectExitCode(0)

	// Get the first commitID we find, which should be the first commit for the project
	snapshot := cp.TrimmedSnapshot()
	commitRe := regexp.MustCompile(`[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}`)
	firstCommit := commitRe.FindString(snapshot)

	if firstCommit == "" {
		suite.FailNow("Could not match commitID against output:\n" + snapshot)
	}

	suite.Run("install", func() {
		cp := ts.Spawn("install", "urllib3@1.25.6")
		cp.ExpectRe("(?:Package added|project is currently building)", 30*time.Second)
		cp.Wait()
	})

	suite.Run("install (update)", func() {
		cp := ts.Spawn("install", "urllib3@1.25.8")
		cp.ExpectRe("(?:Package updated|project is currently building)", 30*time.Second)
		cp.Wait()
	})

	suite.Run("uninstall", func() {
		cp := ts.Spawn("uninstall", "urllib3")
		cp.ExpectRe("(?:Package uninstalled|project is currently building)", 30*time.Second)
		cp.Wait()
	})

	cp = ts.Spawn("revert", firstCommit)
	cp.Send("y")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("pull")
	cp.ExpectExitCode(0)

	// expecting json output, as table wraps message in column
	cp = ts.Spawn("history", "--output=json")
	cp.ExpectLongString(fmt.Sprintf("Reverting to commit %s", firstCommit))
	cp.ExpectExitCode(0)
}

func (suite *PackageIntegrationTestSuite) PrepareActiveStateYAML(ts *e2e.Session) {
	asyData := `project: "https://platform.activestate.com/ActiveState-CLI/List?commitID=a9d0bc88-585a-49cf-89c1-6c07af781cff"
scripts:
  - name: test-pyparsing
    language: python3
    value: |
      from pyparsing import Word, alphas
      print(Word(alphas).parseString("TEST"))
`
	ts.PrepareActiveStateYAML(asyData)
}

func TestPackageIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(PackageIntegrationTestSuite))
}
