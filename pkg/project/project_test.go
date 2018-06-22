package project

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

func loadProject(t *testing.T, contentType string) *projectfile.Project {
	project := &projectfile.Project{}
	contentsGeneral := strings.TrimSpace(`
platforms:
  - name: Windows
    os: windows
  - name: Linux
    os: linux
  - name: OSX
    os: darwin
name: foo
environments: "something"
version: "1.0"
namespace: "my/name/space"
owner: "carey"
`)
	contentsVars := strings.TrimSpace(`
platforms:
  - name: Windows
    os: windows
  - name: Linux
    os: linux
  - name: OSX
    os: darwin
variables:
  - name: bar
    value: baz
    constraints:
      platform: Linux
  - name: bar
    value: quux
    constraints:
      platform: Windows
  - name: bar
    value: "1.3"
    constraints:
      platform: OSX
`)
	contentsHooks := strings.TrimSpace(`
platforms:
  - name: Windows
    os: windows
  - name: Linux
    os: linux
  - name: OSX
    os: darwin
hooks:
  - name: bar
    value: baz
    constraints:
      platform: Linux
  - name: baz
    value: quux
    constraints:
      platform: Windows
  - name: gonzo
    value: "1.3"
    constraints:
      platform: OSX
`)
	contentsCmds := strings.TrimSpace(`
platforms:
  - name: Windows
    os: windows
  - name: Linux
    os: linux
  - name: OSX
    os: darwin
commands:
  - name: bar
    value: baz
    constraints:
      platform: Linux
  - name: bar
    value: quux
    constraints:
      platform: Windows
  - name: bar
    value: "1.3"
    constraints:
      platform: OSX
`)
	contentsLangs := strings.TrimSpace(`
platforms:
  - name: Windows
    os: windows
  - name: Linux
    os: linux
  - name: OSX
    os: darwin
languages:
  - name: bar
    version: "1.1"
    constraints:
      platform: Linux
  - name: baz
    version: "1.2"
    constraints:
      platform: Windows
  - name: quiznar
    version: "1.3"
    constraints:
      platform: OSX
`)
	contentsPkgs := strings.TrimSpace(`
platforms:
  - name: Windows
    os: windows
  - name: Linux
    os: linux
  - name: OSX
    os: darwin
languages:
  - name: foo
    version: "1.0"
    packages:
      - name: foo
        version: "1.1"
        constraints:
          platform: Linux
      - name: bar
        version: "1.2"
        constraints:
          platform: Windows
      - name: baz
        version: "1.3"
        constraints:
          platform: OSX
`)
	var contents map[string]string
	contents = make(map[string]string)
	contents["Vars"] = contentsVars
	contents["Langs"] = contentsLangs
	contents["Cmds"] = contentsCmds
	contents["Pkgs"] = contentsPkgs
	contents["Hooks"] = contentsHooks
	contents["General"] = contentsGeneral
	fail := yaml.Unmarshal([]byte(contents[contentType]), project)
	assert.Nil(t, fail, "Unmarshalled YAML")
	project.Persist()

	return project
}

func TestGet(t *testing.T) {
	loadProject(t, "General")
	val := Get()
	assert.IsType(t, &Project{}, val, "Should be a project.go.Project")
}

func TestGetSafe(t *testing.T) {
	loadProject(t, "General")
	val, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	assert.IsType(t, &Project{}, val, "Should be a project.go.Project")
}

func TestName(t *testing.T) {
	loadProject(t, "General")
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	assert.Equal(t, "foo", prj.Name(), "Values should match")
}

func TestOwner(t *testing.T) {
	loadProject(t, "General")
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	assert.Equal(t, "carey", prj.Owner(), "Values should match")
}

func TestNamespace(t *testing.T) {
	loadProject(t, "General")
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	assert.Equal(t, "my/name/space", prj.Namespace(), "Values should match")
}

func TestEnvironment(t *testing.T) {
	loadProject(t, "General")
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	assert.Equal(t, "something", prj.Environments(), "Values should match")
}

func TestVersion(t *testing.T) {
	loadProject(t, "General")
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	assert.Equal(t, "1.0", prj.Version(), "Values should match")
}

func TestPlatforms(t *testing.T) {
	loadProject(t, "General")
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	val := prj.Platforms()
	assert.Equal(t, 3, len(val), "Values should match")
}

func TestHooks(t *testing.T) {
	loadProject(t, "Hooks")
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	hooks := prj.Hooks()
	assert.Equal(t, 1, len(hooks), "Should match 1 out of three constrained items")
}

func TestLanguages(t *testing.T) {
	loadProject(t, "Langs")
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	languages := prj.Languages()
	assert.Equal(t, 1, len(languages), "Should match 1 out of three constrained items")
}

func TestPackages(t *testing.T) {
	loadProject(t, "Pkgs")
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	languages := prj.Languages()
	language := languages[0]
	packages, fail := language.Packages()
	assert.Nil(t, fail, "Run without failure")
	assert.Equal(t, 1, len(packages), "Should match 1 out of three constrained items")
}

func TestCommands(t *testing.T) {
	loadProject(t, "Cmds")
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	commands := prj.Commands()
	assert.Equal(t, 1, len(commands), "Should match 1 out of three constrained items")
}

func TestVariables(t *testing.T) {
	loadProject(t, "Vars")
	prj, fail := GetSafe()
	assert.Nil(t, fail, "Run without failure")
	variables := prj.Variables()
	assert.Equal(t, 1, len(variables), "Should match 1 out of three constrained items")
}
