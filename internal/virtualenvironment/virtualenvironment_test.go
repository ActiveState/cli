package virtualenvironment

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/pkg/projectfile"
	yaml "gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	os.Chdir(filepath.Join(root, "test"))

	datadir := config.GetDataDir()
	os.RemoveAll(filepath.Join(datadir, "virtual"))
	os.RemoveAll(filepath.Join(datadir, "packages"))
	os.RemoveAll(filepath.Join(datadir, "languages"))

	venvs = make(map[string]VirtualEnvironmenter)
}

func teardown() {
	projectfile.Reset()
}

func TestActivate(t *testing.T) {
	setup(t)

	fail := Activate()
	if runtime.GOOS != "windows" {
		assert.NoError(t, fail.ToError(), "Should activate")
	} else {
		// Since creating symlinks on Windows requires admin privilages for now,
		// test activation should fail.
		assert.Error(t, fail, "Symlinking requires admin privilages for now")
	}

	setup(t)
	project := &projectfile.Project{}
	dat := strings.TrimSpace(`
		name: valueForName
		owner: valueForOwner`)
	yaml.Unmarshal([]byte(dat), &project)
	project.Persist()

	fail = Activate()
	if runtime.GOOS != "windows" {
		assert.NoError(t, fail.ToError(), "Should activate, even if no languages are defined")
	} else {
		// Since creating symlinks on Windows requires admin privilages for now,
		// test activation should fail.
		assert.Error(t, fail, "Symlinking requires admin privilages for now")
	}

	setup(t)
	project = &projectfile.Project{}
	dat = strings.TrimSpace(`
		name: valueForName
		owner: valueForOwner
		languages:
		- name: Python
		version: 2.7.12`)
	yaml.Unmarshal([]byte(dat), &project)
	project.Persist()

	fail = Activate()
	if runtime.GOOS != "windows" {
		assert.NoError(t, fail.ToError(), "Should activate, even if no packages are defined")
	} else {
		// Since creating symlinks on Windows requires admin privilages for now,
		// test activation should fail.
		assert.Error(t, fail, "Symlinking requires admin privilages for now")
	}

	teardown()
}

func TestActivateFailureUnknownLanguage(t *testing.T) {
	setup(t)

	project := projectfile.Get()
	language := projectfile.Language{Name: "foo"}
	project.Languages = append(project.Languages, language)
	project.Persist()

	err := Activate()
	assert.Error(t, err, "Should not activate due to unknown language")

	teardown()
}
