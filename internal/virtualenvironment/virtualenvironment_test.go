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

	venvs = make(map[string]VirtualEnvironment)
}

func TestActivate(t *testing.T) {
	setup(t)

	project, err := projectfile.Get()
	assert.NoError(t, err, "Should get project file")

	err = Activate(project)
	assert.NoError(t, err, "Should activate")

	setup(t)
	project = &projectfile.Project{}
	dat := strings.TrimSpace(`
name: valueForName
owner: valueForOwner`)
	yaml.Unmarshal([]byte(dat), &project)

	err = Activate(project)
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

	err = Activate(project)
	assert.NoError(t, err, "Should activate, even if no packages are defined")
}

func TestActivateFailureUnknownLanguage(t *testing.T) {
	setup(t)

	project, err := projectfile.Get()
	assert.NoError(t, err, "Should get project file")

	language := projectfile.Language{Name: "foo"}

	project.Languages = append(project.Languages, language)

	err = Activate(project)
	assert.Error(t, err, "Should not activate due to unknown language")
}

func TestGetEnv(t *testing.T) {
	setup(t)

	project, err := projectfile.Get()

	assert.NoError(t, err, "Should get project file")

	_, err = GetEnv(project, &project.Languages[0])
	assert.Error(t, err, "Should fail due to missing directory")

	setup(t)
	createFolderStructure(project)

	_, err = GetEnv(project, &project.Languages[0])
	assert.NoError(t, err, "Should get venv")

	// Calling it again for the cached version
	_, err = GetEnv(project, &project.Languages[0])
	assert.NoError(t, err, "Should get venv")
}

func TestActivateLanguageVenv(t *testing.T) {
	setup(t)

	project, _ := projectfile.Get()

	venv := &python.VirtualEnvironment{}

	err := ActivateLanguageVenv(project, &project.Languages[0], venv)
	assert.Error(t, err, "Should fail to activate venv because target folder doesnt exist")

	createFolderStructure(project)
	err = ActivateLanguageVenv(project, &project.Languages[0], venv)
	assert.NoError(t, err, "Should activate the venv")
}

func TestLoadLanguage(t *testing.T) {
	setup(t)

	project, _ := projectfile.Get()
	language := &project.Languages[0]

	venv := &python.VirtualEnvironment{}

	datadir := config.GetDataDir()
	datadir = filepath.Join(datadir, "virtual", project.Owner, project.Name, language.Name, language.Version)

	venv.SetProject(project)
	venv.SetLanguageMeta(language)
	venv.SetDataDir(datadir)

	err := loadLanguage(project, language, venv)
	assert.Error(t, err, "Should fail to load language because target folder doesnt exist")

	createFolderStructure(project)
	err = loadLanguage(project, language, venv)
	assert.NoError(t, err, "Should load the language")
}

func TestGetHashFromLanguage(t *testing.T) {
	setup(t)

	project, _ := projectfile.Get()
	language := &project.Languages[0]

	hash := getHashFromLanguage(language)
	assert.NotEmpty(t, hash, "Hash should be set")
}

func TestObtainLanguage(t *testing.T) {
	setup(t)

	project, _ := projectfile.Get()
	language := &project.Languages[0]

	path, err := obtainLanguage(language)
	assert.NoError(t, err, "Should obtain language")
	assert.NotEmpty(t, path, "Should return language path")
}

func TestLoadPackage(t *testing.T) {
	setup(t)

	project, _ := projectfile.Get()
	language := &project.Languages[0]
	pkg := &language.Packages[0]

	venv := &python.VirtualEnvironment{}

	datadir := config.GetDataDir()
	datadir = filepath.Join(datadir, "virtual", project.Owner, project.Name, language.Name, language.Version)

	venv.SetProject(project)
	venv.SetLanguageMeta(language)
	venv.SetDataDir(datadir)

	err := loadPackage(project, language, pkg, venv)
	assert.Error(t, err, "Should fail to load package because target folder doesnt exist")

	createFolderStructure(project)
	err = loadPackage(project, language, pkg, venv)
	assert.NoError(t, err, "Should load the package")
}

func TestGetHashFromPackage(t *testing.T) {
	setup(t)

	project, _ := projectfile.Get()
	language := &project.Languages[0]
	pkg := &language.Packages[0]

	hash := getHashFromPackage(pkg)
	assert.NotEmpty(t, hash, "Hash should be set")
}

func TestObtainPackage(t *testing.T) {
	setup(t)

	project, _ := projectfile.Get()
	language := &project.Languages[0]
	pkg := &language.Packages[0]

	path, err := obtainPackage(language, pkg)
	assert.NoError(t, err, "Should obtain language")
	assert.NotEmpty(t, path, "Should return package path")
}

func TestCreateFolderStructure(t *testing.T) {
	setup(t)

	project, _ := projectfile.Get()

	err := createFolderStructure(project)
	assert.NoError(t, err, "Creates folder structure")
}
