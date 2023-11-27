package integration

import (
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

func TestImportIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(ImportIntegrationTestSuite))
}
