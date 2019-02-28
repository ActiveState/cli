package python

import (
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/artifact"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/distribution"
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

	err = os.Chdir(suite.testDir)
	suite.Require().NoError(err, "Should change dir")
}

func (suite *PythonTestSuite) AfterTest(suiteName, testName string) {
	root, err := environment.GetRootPath()
	suite.Require().NoError(err, "Should fetch cwd")
	os.Chdir(root)

	datadir := config.GetDataDir()
	os.RemoveAll(filepath.Join(datadir, "virtual"))
	os.RemoveAll(filepath.Join(datadir, "packages"))
	os.RemoveAll(filepath.Join(datadir, "languages"))
	os.RemoveAll(filepath.Join(datadir, "artifacts"))
}

func (suite *PythonTestSuite) TestLanguage() {
	venv := &VirtualEnvironment{}
	suite.Equal("python3", venv.Language(), "Should return python")

	venv.SetArtifact(&artifact.Artifact{
		Meta: &artifact.Meta{
			Name: "python2",
		},
		Path: "test",
	})
	suite.Equal("python2", venv.Language(), "Should return python")
}

func (suite *PythonTestSuite) TestDataDir() {
	venv := &VirtualEnvironment{}
	suite.Empty(venv.DataDir())

	venv.SetDataDir("/foo")
	suite.NotEmpty(venv.DataDir(), "Should set the datadir")
}

func (suite *PythonTestSuite) TestLanguageMeta() {
	venv := &VirtualEnvironment{}
	suite.Nil(venv.Artifact(), "Should not have artifact info")

	venv.SetArtifact(&artifact.Artifact{
		Meta: &artifact.Meta{
			Name: "test",
		},
		Path: "test",
	})
	suite.NotNil(venv.Artifact(), "Should have artifact info")
}

func (suite *PythonTestSuite) TestLoadPackageFromPath() {
	venv := &VirtualEnvironment{}

	datadir := filepath.Join(os.TempDir(), "as-state-test")
	os.RemoveAll(datadir)
	os.Mkdir(datadir, os.ModePerm)
	venv.SetDataDir(datadir)

	dist, fail := distribution.Obtain()
	suite.Require().NoError(fail.ToError())

	var language *artifact.Artifact
	for _, lang := range dist.Languages {
		if lang.Meta.Name == venv.Language() {
			language = lang
			break
		}
	}

	fail = venv.LoadArtifact(language)
	if runtime.GOOS != "windows" {
		suite.Require().NoError(fail.ToError(), "Loads artifact without errors")
	} else {
		// Since creating symlinks on Windows requires admin privilages for now,
		// artifacts should not load correctly.
		suite.Error(fail, "Symlinking requires admin privilages for now")
	}
	artf := dist.Artifacts[language.Hash][0]
	// Manually generate expect home where packages will be linked
	langPkgDir := filepath.Join(datadir, "language", "lib", "python2.7", "site-packages")
	os.MkdirAll(langPkgDir, os.ModePerm)

	fail = venv.LoadArtifact(artf)
	if runtime.GOOS != "windows" {
		suite.Require().NoError(fail.ToError(), "Loads artifact without errors")
	} else {
		// Since creating symlinks on Windows requires admin privilages for now,
		// artifacts should not load correctly.
		suite.Error(fail, "Symlinking requires admin privilages for now")
	}

	// Todo: Test with datadir as source, not the archived version
	if runtime.GOOS != "windows" {
		suite.FileExists(filepath.Join(langPkgDir, artf.Meta.Name, "artifact.json"), "Should create a package symlink")
	} else {
		// Since creating symlinks on Windows requires admin privilages for now,
		// the symlinked file should not exist.  Check if it was created or not. Skip if not.
		_, err := os.Stat(filepath.Join(datadir, "language", "Lib", "site-packages", artf.Meta.Name, "artifact.json"))
		if err == nil {
			suite.FileExists(filepath.Join(datadir, "language", "Lib", "site-packages", artf.Meta.Name, "artifact.json"), "Should create a package symlink")
		}
	}
}

func (suite *PythonTestSuite) TestActivate() {
	venv := &VirtualEnvironment{}

	venv.SetArtifact(&artifact.Artifact{
		Meta: &artifact.Meta{
			Name:    "python",
			Version: "2.7.11",
		},
		Path: "test",
	})

	datadir := config.GetDataDir()
	datadir = filepath.Join(datadir, "test")
	venv.SetDataDir(datadir)

	venv.Activate()

	suite.DirExists(filepath.Join(venv.DataDir(), "bin"))
	suite.DirExists(filepath.Join(venv.DataDir(), "lib"))
}

func (suite *PythonTestSuite) TestEnv_NoPythonDirOrDistsInstalled() {
	venv := &VirtualEnvironment{}
	dataDir := path.Join(suite.testDir, "venv-python3-empty")
	venv.SetDataDir(dataDir)
	suite.Equal(map[string]string{}, venv.Env())
}

func (suite *PythonTestSuite) TestEnv_NoDistsInstalled() {
	venv := &VirtualEnvironment{}
	dataDir := path.Join(suite.testDir, "venv-python3-nodist")
	venv.SetDataDir(dataDir)
	suite.Equal(map[string]string{}, venv.Env())
}

func (suite *PythonTestSuite) TestEnv_WithDistsInstalled() {
	venv := &VirtualEnvironment{}
	dataDir := path.Join(suite.testDir, "venv-python3")
	venv.SetDataDir(dataDir)
	suite.Equal(map[string]string{
		"PATH": path.Join(dataDir, "python", "apy-1.2.3-linux-glibc", "bin"),
	}, venv.Env())
}

func Test_PythonTestSuite(t *testing.T) {
	suite.Run(t, new(PythonTestSuite))
}
