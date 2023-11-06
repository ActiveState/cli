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

func (suite *ImportIntegrationTestSuite) TestImport() {
	suite.OnlyRunForTags(tagsuite.Import)
	ts := e2e.New(suite.T(), false)
	defer ts.Close()

	username, _ := ts.CreateNewUser()
	namespace := fmt.Sprintf("%s/%s", username, "Python3")

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
	ts.IgnoreLogErrors()
}

func TestImportIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ImportIntegrationTestSuite))
}
