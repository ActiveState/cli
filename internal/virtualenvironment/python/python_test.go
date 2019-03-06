package python_test

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/installer/mock"
	"github.com/ActiveState/cli/internal/virtualenvironment/python"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
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
	pjfile.Languages = append(pjfile.Languages, projectfile.Language{Name: "Python", Version: "3"})
	pjfile.Persist()

	cwd, err := environment.GetRootPath()
	suite.Require().NoError(err, "unable to obtain the cwd")

	suite.testDir = filepath.Join(cwd, "internal", "virtualenvironment", "python", "testdata")
	fileutils.MkdirUnlessExists(suite.testDir)

	suite.dataDir, err = ioutil.TempDir("", "venv-python3-empty")
	suite.Require().NoError(err, "creating temp data dir")

	suite.Require().NoError(os.Chdir(suite.testDir), "Should change dir")
}

func (suite *PythonTestSuite) AfterTest(suiteName, testName string) {
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "Should fetch cwd")
	os.Chdir(root)
}

func (suite *PythonTestSuite) TestNew_InstallerNil() {
	venv, failure := python.NewVirtualEnvironment("/tmp", nil)
	suite.Require().Nil(venv, "python venv should be nil")
	suite.Require().NotNil(failure)
	suite.Equal(failures.FailVerify, failure.Type)
	suite.Equal(locale.Tr("venv_installer_is_nil"), failure.Error())
}

func (suite *PythonTestSuite) TestLanguage() {
	venv, failure := python.NewVirtualEnvironment("/tmp", mock.NewMockInstaller())
	suite.Require().Nil(failure)
	suite.Equal("python3", venv.Language(), "Should return python")
}

func (suite *PythonTestSuite) TestDataDir() {
	venv, failure := python.NewVirtualEnvironment("/foo", mock.NewMockInstaller())
	suite.Require().Nil(failure)
	suite.Equal("/foo", venv.DataDir(), "Should set the data-dir")
}

func (suite *PythonTestSuite) TestWorkingDir() {
	venv, failure := python.NewVirtualEnvironment("/foo", mock.NewMockInstaller())
	suite.Require().Nil(failure)
	suite.Zero(venv.WorkingDirectory())
}

func (suite *PythonTestSuite) TestActivate_NoDistInstalled() {
	mockInstaller := mock.NewMockInstaller()
	mockInstaller.On("Install").Return(nil)

	venv, failure := python.NewVirtualEnvironment(suite.dataDir, mockInstaller)
	suite.Require().Nil(failure)
	suite.Nil(venv.Activate())

	mockInstaller.AssertCalled(suite.T(), "Install")
}

func (suite *PythonTestSuite) TestActivate_WithDistInstalled() {
	mockInstaller := mock.NewMockInstaller()

	dataDir := path.Join(suite.testDir, "venv-python3")
	venv, failure := python.NewVirtualEnvironment(dataDir, mockInstaller)
	suite.Require().Nil(failure)
	suite.Nil(venv.Activate())

	mockInstaller.AssertNotCalled(suite.T(), "Install")
}

func (suite *PythonTestSuite) TestEnv_NoDistInstalled() {
	venv, failure := python.NewVirtualEnvironment(suite.dataDir, mock.NewMockInstaller())
	suite.Require().Nil(failure)
	suite.Equal(map[string]string{
		"PATH": path.Join(suite.dataDir, "bin"),
	}, venv.Env())
}

func (suite *PythonTestSuite) TestEnv_WithDistInstalled() {
	dataDir := path.Join(suite.testDir, "venv-python3")
	venv, failure := python.NewVirtualEnvironment(dataDir, mock.NewMockInstaller())
	suite.Require().Nil(failure)
	suite.Equal(map[string]string{
		"PATH": path.Join(dataDir, "bin"),
	}, venv.Env())
}

func Test_PythonTestSuite(t *testing.T) {
	suite.Run(t, new(PythonTestSuite))
}
