package python

import (
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/suite"
)

type PythonTestSuite struct {
	suite.Suite

	testDir string
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

	suite.Require().NoError(os.Chdir(suite.testDir), "Should change dir")
}

func (suite *PythonTestSuite) AfterTest(suiteName, testName string) {
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "Should fetch cwd")
	os.Chdir(root)
}

func (suite *PythonTestSuite) TestLanguage() {
	venv := NewVirtualEnvironment("/tmp", "/cache")
	suite.Equal("python3", venv.Language(), "Should return python")
}

func (suite *PythonTestSuite) TestDirs() {
	venv := NewVirtualEnvironment("/foo", "/bar")
	suite.Equal("/foo", venv.DataDir(), "Should set the data-dir")
	suite.Equal("/bar", venv.CacheDir(), "Should set the cache-dir")
}

func (suite *PythonTestSuite) TestActivate() {
	dataDir := filepath.Join(config.GetDataDir(), "test")
	cacheDir := filepath.Join(config.GetCacheDir(), "test")
	venv := NewVirtualEnvironment(dataDir, cacheDir)
	venv.Activate()
	suite.DirExists(filepath.Join(venv.DataDir(), "bin"))
	suite.DirExists(filepath.Join(venv.DataDir(), "lib"))
}

func (suite *PythonTestSuite) TestEnv_NoDistsInstalled() {
	cacheDir := path.Join(suite.testDir, "venv-python3-empty")
	venv := NewVirtualEnvironment(suite.testDir, cacheDir)
	suite.Equal(map[string]string{
		"PATH": path.Join(cacheDir, "bin") + string(os.PathListSeparator) + path.Join(suite.testDir, "bin"),
	}, venv.Env())
}

func (suite *PythonTestSuite) TestEnv_WithDistInstalled() {
	cacheDir := path.Join(suite.testDir, "venv-python3")
	venv := NewVirtualEnvironment(suite.testDir, cacheDir)
	suite.Equal(map[string]string{
		"PATH": path.Join(cacheDir, "bin") + string(os.PathListSeparator) + path.Join(suite.testDir, "bin"),
	}, venv.Env())
}

func Test_PythonTestSuite(t *testing.T) {
	suite.Run(t, new(PythonTestSuite))
}
