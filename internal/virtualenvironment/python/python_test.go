package python

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
)

func setup(t *testing.T) {
	root, _ := environment.GetRootPath()
	os.Chdir(filepath.Join(root, "test"))
}

func TestLanguage(t *testing.T) {
	venv := &VirtualEnvironment{}
	assert.Equal(t, "Python", venv.Language(), "Should return Python")
}

func TestDataDir(t *testing.T) {
	venv := &VirtualEnvironment{}
	assert.Empty(t, venv.DataDir())

	venv.SetDataDir("/foo")
	assert.NotEmpty(t, venv.DataDir(), "Should set the datadir")
}

func TestLanguageMeta(t *testing.T) {
	setup(t)

	project := projectfile.Get()
	language := &project.Languages[0]

	venv := &VirtualEnvironment{}
	assert.Nil(t, venv.LanguageMeta(), "Should not have language meta")

	venv.SetLanguageMeta(language)
	assert.NotNil(t, venv.LanguageMeta(), "Should have language meta")
}

func TestLoadLanguageFromPath(t *testing.T) {
	root, _ := environment.GetRootPath()
	venv := &VirtualEnvironment{}

	source := filepath.Join(root, "test", "builder", "python", "2.7.12")

	datadir := filepath.Join(os.TempDir(), "as-state-test")
	os.RemoveAll(datadir)
	os.Mkdir(datadir, os.ModePerm)
	venv.SetDataDir(datadir)

	venv.LoadLanguageFromPath(source)

	assert.FileExists(t, filepath.Join(datadir, "language"), "Should create a language symlink")
}

func TestLoadPackageFromPath(t *testing.T) {
	root, _ := environment.GetRootPath()
	venv := &VirtualEnvironment{}
	pkg := &projectfile.Package{Name: "peewee"}

	source := filepath.Join(root, "test", "builder", "python", "2.7.12", "peewee")

	datadir := filepath.Join(os.TempDir(), "as-state-test")
	os.RemoveAll(datadir)
	os.Mkdir(datadir, os.ModePerm)
	venv.SetDataDir(datadir)

	venv.LoadPackageFromPath(source, pkg)

	// Todo: Test with datadir as source, not the archived version
	assert.FileExists(t, filepath.Join(datadir, "lib", "2.9.1.tar.gz"), "Should create a package symlink")
}

func TestActivate(t *testing.T) {
	setup(t)

	project := projectfile.Get()
	language := &project.Languages[0]

	venv := &VirtualEnvironment{}

	venv.SetLanguageMeta(language)
	venv.SetDataDir("")

	os.Setenv("PYTHONPATH", "")
	os.Setenv("PATH", "")

	venv.Activate()

	assert.NotEmpty(t, os.Getenv("PYTHONPATH"), "PYTHONPATH should be set")
	assert.NotEmpty(t, os.Getenv("PATH"), "PATH should be set")
}
