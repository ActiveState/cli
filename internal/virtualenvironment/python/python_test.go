package python

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/cli/internal/artifact"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/distribution"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func setup(t *testing.T) {
	pjfile := projectfile.Project{}
	pjfile.Languages = append(pjfile.Languages, projectfile.Language{Name: "Python", Version: "2"})
	pjfile.Languages = append(pjfile.Languages, projectfile.Language{Name: "Python", Version: "3"})
	pjfile.Persist()
	cwd, err := environment.GetRootPath()
	assert.NoError(t, err, "Should fetch cwd")
	testDir := filepath.Join(cwd, "internal", "virtualenvironment", "python", "testdata")
	os.Mkdir(testDir, os.ModePerm) // For now there is nothing in the testdata dir so it's not cloned.  Don't care if it errors out.
	err = os.Chdir(testDir)
	assert.NoError(t, err, "Should change dir")
}

func teardown(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should fetch cwd")
	os.Chdir(root)

	datadir := config.GetDataDir()
	os.RemoveAll(filepath.Join(datadir, "virtual"))
	os.RemoveAll(filepath.Join(datadir, "packages"))
	os.RemoveAll(filepath.Join(datadir, "languages"))
	os.RemoveAll(filepath.Join(datadir, "artifacts"))
}

func TestLanguage(t *testing.T) {
	venv := &VirtualEnvironment{}
	assert.Equal(t, "python3", venv.Language(), "Should return python")

	venv.SetArtifact(&artifact.Artifact{
		Meta: &artifact.Meta{
			Name: "python2",
		},
		Path: "test",
	})
	assert.Equal(t, "python2", venv.Language(), "Should return python")
	teardown(t)
}

func TestDataDir(t *testing.T) {
	venv := &VirtualEnvironment{}
	assert.Empty(t, venv.DataDir())

	venv.SetDataDir("/foo")
	assert.NotEmpty(t, venv.DataDir(), "Should set the datadir")
	teardown(t)
}

func TestLanguageMeta(t *testing.T) {
	setup(t)

	venv := &VirtualEnvironment{}
	assert.Nil(t, venv.Artifact(), "Should not have artifact info")

	venv.SetArtifact(&artifact.Artifact{
		Meta: &artifact.Meta{
			Name: "test",
		},
		Path: "test",
	})
	assert.NotNil(t, venv.Artifact(), "Should have artifact info")
	teardown(t)
}

func TestLoadPackageFromPath(t *testing.T) {
	setup(t)

	venv := &VirtualEnvironment{}

	datadir := filepath.Join(os.TempDir(), "as-state-test")
	os.RemoveAll(datadir)
	os.Mkdir(datadir, os.ModePerm)
	venv.SetDataDir(datadir)

	dist, fail := distribution.Obtain()
	assert.NoError(t, fail.ToError())

	var language *artifact.Artifact
	for _, lang := range dist.Languages {
		if lang.Meta.Name == venv.Language() {
			language = lang
			break
		}
	}

	fail = venv.LoadArtifact(language)
	if runtime.GOOS != "windows" {
		assert.NoError(t, fail.ToError(), "Loads artifact without errors")
	} else {
		// Since creating symlinks on Windows requires admin privilages for now,
		// artifacts should not load correctly.
		assert.Error(t, fail, "Symlinking requires admin privilages for now")
	}
	artf := dist.Artifacts[language.Hash][0]
	// Manually generate expect home where packages will be linked
	langPkgDir := filepath.Join(datadir, "language", "lib", "python2.7", "site-packages")
	os.MkdirAll(langPkgDir, os.ModePerm)

	fail = venv.LoadArtifact(artf)
	if runtime.GOOS != "windows" {
		assert.NoError(t, fail.ToError(), "Loads artifact without errors")
	} else {
		// Since creating symlinks on Windows requires admin privilages for now,
		// artifacts should not load correctly.
		assert.Error(t, fail, "Symlinking requires admin privilages for now")
	}

	// Todo: Test with datadir as source, not the archived version
	if runtime.GOOS != "windows" {
		assert.FileExists(t, filepath.Join(langPkgDir, artf.Meta.Name, "artifact.json"), "Should create a package symlink")
	} else {
		// Since creating symlinks on Windows requires admin privilages for now,
		// the symlinked file should not exist.  Check if it was created or not. Skip if not.
		_, err := os.Stat(filepath.Join(datadir, "language", "Lib", "site-packages", artf.Meta.Name, "artifact.json"))
		if err == nil {
			assert.FileExists(t, filepath.Join(datadir, "language", "Lib", "site-packages", artf.Meta.Name, "artifact.json"), "Should create a package symlink")
		}
	}
	teardown(t)
}

func TestActivate(t *testing.T) {
	setup(t)

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

	assert.DirExists(t, filepath.Join(venv.DataDir(), "bin"))
	assert.DirExists(t, filepath.Join(venv.DataDir(), "lib"))
	teardown(t)
}
