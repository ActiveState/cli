package python

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/suite"
)

type PythonTestSuite struct {
	suite.Suite

	testDir string
	dataDir string
}

func (suite *PythonTestSuite) BeforeTest(suiteName, testName string) {
	pjfile := projectfile.Project{}
	pjfile.Languages = append(pjfile.Languages, projectfile.Language{Name: "Python", Version: "2"})
	pjfile.Languages = append(pjfile.Languages, projectfile.Language{Name: "Python", Version: "3"})
	pjfile.Persist()

	cwd, err := environment.GetRootPath()
	suite.Require().NoError(err, "unable to obtain the cwd")

	suite.testDir = filepath.Join(cwd, "internal", "virtualenvironment", "python", "testdata")
	fileutils.MkdirUnlessExists(suite.testDir)
	suite.dataDir = path.Join(suite.testDir, "venv-python3-empty")

	suite.Require().NoError(os.Chdir(suite.testDir), "Should change dir")
}

func (suite *PythonTestSuite) AfterTest(suiteName, testName string) {
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "Should fetch cwd")
	os.Chdir(root)
}

func (suite *PythonTestSuite) TestLanguage() {
	venv := NewVirtualEnvironment("/tmp")
	suite.Equal("python3", venv.Language(), "Should return python")
}

func (suite *PythonTestSuite) TestDirs() {
	venv := NewVirtualEnvironment("/foo")
	suite.Equal("/foo", venv.DataDir(), "Should set the data-dir")
}

func (suite *PythonTestSuite) TestActivate() {
	venv := NewVirtualEnvironment(suite.dataDir)
	suite.Nil(venv.Activate())
}

func (suite *PythonTestSuite) TestEnv_NoDistsInstalled() {
	venv := NewVirtualEnvironment(suite.dataDir)
	suite.Equal(map[string]string{
		"PATH": path.Join(suite.dataDir, "bin"),
	}, venv.Env())
}

func (suite *PythonTestSuite) TestEnv_WithDistInstalled() {
	dataDir := path.Join(suite.testDir, "venv-python3")
	venv := NewVirtualEnvironment(dataDir)
	suite.Equal(map[string]string{
		"PATH": path.Join(dataDir, "bin"),
	}, venv.Env())
}

func Test_PythonTestSuite(t *testing.T) {
	suite.Run(t, new(PythonTestSuite))
}
