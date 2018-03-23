package virtualenvironment

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
	yaml "gopkg.in/yaml.v2"

	"github.com/ActiveState/ActiveState-CLI/internal/config"
	"github.com/ActiveState/ActiveState-CLI/internal/environment"
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
	assert.NoError(t, fail.ToError(), "Should activate")

	setup(t)
	project := &projectfile.Project{}
	dat := strings.TrimSpace(`
		name: valueForName
		owner: valueForOwner`)
	yaml.Unmarshal([]byte(dat), &project)
	project.Persist()

	fail = Activate()
	assert.NoError(t, fail.ToError(), "Should activate, even if no languages are defined")

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
	assert.NoError(t, fail.ToError(), "Should activate, even if no packages are defined")

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
