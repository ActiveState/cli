package integration

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/e2e"
	"github.com/ActiveState/cli/internal/testhelpers/tagsuite"
	"github.com/stretchr/testify/suite"
)

type ImportIntegrationTestSuite struct {
	tagsuite.Suite
}

func (suite *ImportIntegrationTestSuite) TestImport_detached() {
	suite.OnlyRunForTags(tagsuite.Import)
	if runtime.GOOS == "darwin" {
		suite.T().Skip("Skipping mac for now as the builds are still too unreliable")
		return
	}

	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	cp := ts.Spawn("checkout", "ActiveState-CLI/Python3-Import", ".")
	cp.Expect("Skipping runtime setup")
	cp.Expect("Checked out project")
	cp.ExpectExitCode(0)

	contents := `requests
	urllib3`
	importPath := filepath.Join(ts.Dirs.Work, "requirements.txt")

	err := ioutil.WriteFile(importPath, []byte(strings.TrimSpace(contents)), 0644)
	suite.Require().NoError(err)

	cp = ts.Spawn("import", importPath)
	cp.Expect("Operating on project")
	cp.Expect("ActiveState-CLI/Python3-Import")
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
	cp.Expect("successfully initialized")
	cp.ExpectExitCode(0)

	reqsFilePath := filepath.Join(cp.WorkDirectory(), reqsFileName)

	suite.Run("invalid requirements.txt", func() {
		ts.SetT(suite.T())
		ts.PrepareFile(reqsFilePath, badReqsData)

		cp := ts.Spawn("import", "requirements.txt")
		cp.ExpectNotExitCode(0)
	})

	suite.Run("valid requirements.txt", func() {
		ts.SetT(suite.T())
		ts.PrepareFile(reqsFilePath, reqsData)

		cp := ts.Spawn("import", "requirements.txt")
		cp.ExpectExitCode(0)

		cp = ts.Spawn("push")
		cp.ExpectExitCode(0)

		cp = ts.Spawn("import", "requirements.txt")
		cp.Expect("already exists")
		cp.ExpectNotExitCode(0)
	})

	suite.Run("complex requirements.txt", func() {
		ts.SetT(suite.T())
		ts.PrepareFile(reqsFilePath, complexReqsData)

		cp := ts.Spawn("import", "requirements.txt")
		cp.ExpectExitCode(0)

		cp = ts.Spawn("packages")
		cp.Expect("coverage")
		cp.Expect("docopt")
		cp.Expect("Mopidy-Dirble")
		cp.Expect("requests")
		cp.Expect("Auto") // DX-2272 will change this to 2.30.0
		cp.Expect("urllib3")
		cp.Expect("Auto") // DX-2272 will change this to 1.26.5
		cp.ExpectExitCode(0)
	})
	ts.IgnoreLogErrors()
}

func TestImportIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ImportIntegrationTestSuite))
}
