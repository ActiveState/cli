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
	"github.com/ActiveState/ActiveState-CLI/internal/virtualenvironment/python"
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

	err := Activate()
	assert.NoError(t, err, "Should activate")

	setup(t)
	project := &projectfile.Project{}
	dat := strings.TrimSpace(`
		name: valueForName
		owner: valueForOwner`)
	yaml.Unmarshal([]byte(dat), &project)
	project.Persist()

	err = Activate()
	assert.NoError(t, err, "Should activate, even if no languages are defined")

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

	err = Activate()
	assert.NoError(t, err, "Should activate, even if no packages are defined")

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

func TestGetEnv(t *testing.T) {
	setup(t)

	project := projectfile.Get()

	_, err := GetEnv(&project.Languages[0])
	assert.Error(t, err, "Should fail due to missing directory")

	setup(t)
	createFolderStructure()

	_, err = GetEnv(&project.Languages[0])
	assert.NoError(t, err, "Should get venv")

	// Calling it again for the cached version
	_, err = GetEnv(&project.Languages[0])
	assert.NoError(t, err, "Should get venv")

	teardown()
}

func TestActivateLanguageVenv(t *testing.T) {
	setup(t)

	project := projectfile.Get()

	venv := &python.VirtualEnvironment{}

	err := ActivateLanguageVenv(&project.Languages[0], venv)
	assert.Error(t, err, "Should fail to activate venv because target folder doesnt exist")

	createFolderStructure()
	err = ActivateLanguageVenv(&project.Languages[0], venv)
	assert.NoError(t, err, "Should activate the venv")

	teardown()
}

func TestLoadLanguage(t *testing.T) {
	setup(t)

	project := projectfile.Get()
	language := &project.Languages[0]

	venv := &python.VirtualEnvironment{}

	datadir := config.GetDataDir()
	datadir = filepath.Join(datadir, "virtual", project.Owner, project.Name, language.Name, language.Version)

	venv.SetLanguageMeta(language)
	venv.SetDataDir(datadir)

	err := loadLanguage(language, venv)
	assert.Error(t, err, "Should fail to load language because target folder doesnt exist")

	createFolderStructure()
	err = loadLanguage(language, venv)
	assert.NoError(t, err, "Should load the language")

	teardown()
}

func TestGetHashFromLanguage(t *testing.T) {
	setup(t)

	project := projectfile.Get()
	language := &project.Languages[0]

	hash := getHashFromLanguage(language)
	assert.NotEmpty(t, hash, "Hash should be set")

	teardown()
}

func TestObtainLanguage(t *testing.T) {
	setup(t)

	project := projectfile.Get()
	language := &project.Languages[0]

	path, err := obtainLanguage(language)
	assert.NoError(t, err, "Should obtain language")
	assert.NotEmpty(t, path, "Should return language path")

	teardown()
}

func TestLoadPackage(t *testing.T) {
	setup(t)

	project := projectfile.Get()
	language := &project.Languages[0]
	pkg := &language.Packages[0]

	venv := &python.VirtualEnvironment{}

	datadir := config.GetDataDir()
	datadir = filepath.Join(datadir, "virtual", project.Owner, project.Name, language.Name, language.Version)

	venv.SetLanguageMeta(language)
	venv.SetDataDir(datadir)

	err := loadPackage(language, pkg, venv)
	assert.Error(t, err, "Should fail to load package because target folder doesnt exist")

	createFolderStructure()
	err = loadPackage(language, pkg, venv)
	assert.NoError(t, err, "Should load the package")

	teardown()
}

func TestGetHashFromPackage(t *testing.T) {
	setup(t)

	project := projectfile.Get()
	language := &project.Languages[0]
	pkg := &language.Packages[0]

	hash := getHashFromPackage(pkg)
	assert.NotEmpty(t, hash, "Hash should be set")

	teardown()
}

func TestObtainPackage(t *testing.T) {
	setup(t)

	project := projectfile.Get()
	language := &project.Languages[0]
	pkg := &language.Packages[0]

	path, err := obtainPackage(language, pkg)
	assert.NoError(t, err, "Should obtain language")
	assert.NotEmpty(t, path, "Should return package path")

	teardown()
}

func TestCreateFolderStructure(t *testing.T) {
	setup(t)

	err := createFolderStructure()
	assert.NoError(t, err, "Creates folder structure")

	teardown()
}
