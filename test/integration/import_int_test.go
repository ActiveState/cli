package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ActiveState/termtest"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/internal/testhelpers/suite"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
)

type ImportIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ImportIntegrationTestSuite) TestImport_detached() {
	suite.OnlyRunForTags(tagsuite.Import)

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.PrepareProject("ActiveState-CLI/small-python", "5a1e49e5-8ceb-4a09-b605-ed334474855b")

	contents := `requests
	urllib3`
	importPath := filepath.Join(ts.Dirs.Work, "requirements.txt")

	err := os.WriteFile(importPath, []byte(strings.TrimSpace(contents)), 0644)
	suite.Require().NoError(err)

	ts.LoginAsPersistentUser() // for CVE reporting

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("import", importPath)
	cp.Expect("Operating on project")
	cp.Expect("ActiveState-CLI/small-python")
	cp.Expect("Resolving Dependencies")
	cp.Expect("Import Finished")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("packages")
	cp.Expect("requests")
	cp.Expect("urllib3")
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
packaging==20.3
pyparsing==2.4.6
six==1.14.0
Werkzeug==0.15.6
`
	badReqsData = `Click==7.0
garbage---<<001.X
six==1.14.0
`

	complexReqsData = `coverage!=3.5
docopt>=0.6.1
Mopidy-Dirble>=1.1,<2
requests>=2.2,<2.31.0
urllib3>=1.21.1,<=1.26.5
`
)

func (suite *ImportIntegrationTestSuite) TestImport() {
	suite.OnlyRunForTags(tagsuite.Import)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	user := ts.CreateNewUser()
	namespace := fmt.Sprintf("%s/%s", user.Username, "Python3")

	cp := ts.Spawn("init", "--language", "python", namespace, ts.Dirs.Work)
	cp.Expect("successfully initialized", e2e.RuntimeSourcingTimeoutOpt)
	cp.ExpectExitCode(0)

	reqsFilePath := filepath.Join(cp.WorkDirectory(), reqsFileName)

	suite.Run("invalid requirements.txt", func() {
		ts.SetT(suite.T())
		ts.PrepareFile(reqsFilePath, badReqsData)

		cp = ts.Spawn("import", "requirements.txt")
		cp.ExpectNotExitCode(0)
	})

	suite.Run("valid requirements.txt", func() {
		ts.SetT(suite.T())
		ts.PrepareFile(reqsFilePath, reqsData)

		cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
		cp.ExpectExitCode(0)

		cp = ts.Spawn("import", "requirements.txt")
		cp.ExpectExitCode(0)

		cp = ts.Spawn("import", "requirements.txt")
		cp.Expect("already installed")
		cp.ExpectNotExitCode(0)
	})

	suite.Run("complex requirements.txt", func() {
		ts.SetT(suite.T())
		ts.PrepareFile(reqsFilePath, complexReqsData)

		cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
		cp.ExpectExitCode(0)

		cp = ts.Spawn("import", "requirements.txt")
		cp.ExpectExitCode(0, termtest.OptExpectTimeout(30*time.Second))

		cp = ts.Spawn("packages")
		cp.Expect("coverage")
		cp.Expect("!3.5 → ")
		cp.Expect("docopt")
		cp.Expect(">=0.6.1 →")
		cp.Expect("Mopidy-Dirble")
		cp.Expect("requests")
		cp.Expect(">=2.2,<2.31.0 → 2.30.0")
		cp.Expect("urllib3")
		cp.Expect(">=1.21.1,<=1.26.5 → 1.26.5")
		cp.ExpectExitCode(0)
	})
	ts.IgnoreLogErrors()
}

func (suite *ImportIntegrationTestSuite) TestImportCycloneDx() {
	suite.OnlyRunForTags(tagsuite.Import)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser() // needed to read orgs for private namespace

	ts.PrepareEmptyProject()

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	jsonSbom := filepath.Join(osutil.GetTestDataDir(), "import", "cyclonedx", "bom.json")
	xmlSbom := filepath.Join(osutil.GetTestDataDir(), "import", "cyclonedx", "bom.xml")

	for _, sbom := range []string{jsonSbom, xmlSbom} {
		suite.Run("import "+sbom, func() {
			cp := ts.Spawn("import", sbom)
			cp.Expect("Resolving Dependencies")
			cp.Expect("Failed")
			cp.Expect("unavailable")
			cp.ExpectNotExitCode(0) // solve should fail due to private namespace

			cp = ts.Spawn("history")
			cp.Expect("Import from requirements file")
			cp.Expect("+ body-parser 1.19.0")
			cp.Expect("namespace: private/")
			cp.Expect("+ bytes 3.1.0")
			cp.Expect("namespace: private/")
			cp.ExpectExitCode(0)

			cp = ts.Spawn("reset", "-n")
			cp.ExpectExitCode(0)
		})
	}

	ts.IgnoreLogErrors()
}

func (suite *ImportIntegrationTestSuite) TestImportSpdx() {
	suite.OnlyRunForTags(tagsuite.Import)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	ts.LoginAsPersistentUser() // needed to read orgs for private namespace

	ts.PrepareEmptyProject()

	cp := ts.Spawn("config", "set", constants.AsyncRuntimeConfig, "true")
	cp.ExpectExitCode(0)

	jsonSbom := filepath.Join(osutil.GetTestDataDir(), "import", "spdx", "appbomination.spdx.json")

	cp = ts.Spawn("import", jsonSbom)
	cp.Expect("Resolving Dependencies")
	cp.Expect("Failed")
	cp.Expect("unavailable")
	cp.ExpectNotExitCode(0) // solve should fail due to private namespace

	cp = ts.Spawn("history")
	cp.Expect("Import from requirements file")
	cp.Expect("+ App-BOM-ination 1.0")
	cp.Expect("namespace: private/")
	cp.Expect("+ commons-lang3 3.4")
	cp.Expect("namespace: private/")
	cp.Expect("+ hamcrest-core 1.3")
	cp.Expect("namespace: private/")
	cp.Expect("+ junit 4.12")
	cp.Expect("namespace: private/")
	cp.Expect("+ slf4j-api 1.7.21")
	cp.Expect("namespace: private/")
	cp.ExpectExitCode(0)

	cp = ts.Spawn("reset", "-n")
	cp.ExpectExitCode(0)

	spdxSbom := filepath.Join(osutil.GetTestDataDir(), "import", "spdx", "example1.spdx")

	cp = ts.Spawn("import", spdxSbom)
	cp.Expect("Resolving Dependencies")
	cp.Expect("Failed")
	cp.Expect("unavailable")
	cp.ExpectNotExitCode(0) // solve should fail due to private namespace

	cp = ts.Spawn("history")
	cp.Expect("Import from requirements file")
	cp.Expect("+ hello 1.0.0")
	cp.Expect("namespace: private/")
	cp.ExpectExitCode(0)

	ts.IgnoreLogErrors()
}

func TestImportIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ImportIntegrationTestSuite))
}
