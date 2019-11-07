package integration

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/testhelpers/integration"
	"github.com/ActiveState/cli/pkg/projectfile"
	"gopkg.in/yaml.v2"
)

type EditIntegrationTestSuite struct {
	integration.Suite
}

func (suite *EditIntegrationTestSuite) TestEdit() {
	tempDir, err := ioutil.TempDir("", suite.T().Name())
	suite.Require().NoError(err)

	err = os.Chdir(tempDir)
	suite.Require().NoError(err)

	root := environment.GetRootPathUnsafe()
	editorScript := filepath.Join(root, "test/integration/assets/editor/main.go")
	// suite.SetWd(tempDir)

	fail := fileutils.CopyFile(editorScript, tempDir)
	suite.Require().NoError(fail.ToError())

	contents := strings.TrimSpace(`
project: "https://platform.activestate.com/EditOrg/EditProject?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: test
    value: echo "hello test"
`)

	projectFile := &projectfile.Project{}
	err = yaml.Unmarshal([]byte(contents), projectFile)
	suite.Require().NoError(err, "unexpected error marshalling yaml")

	projectFile.SetPath(filepath.Join(tempDir, constants.ConfigFileName))
	fail = projectFile.Save()
	suite.Require().NoError(err, "should be able to save in temp dir")

	suite.SpawnCustom("go", "build", "main.go")
	suite.AppendEnv([]string{fmt.Sprintf("EDITOR=%s", filepath.Join(tempDir, "main"))})
	suite.Spawn("edit", "test-script")
}

func (suite *EditIntegrationTestSuite) TestEditIntegrationTestSuite(t *testing.T) {
	_ = suite.Run

	integration.RunParallel(t, new(EditIntegrationTestSuite))
}
