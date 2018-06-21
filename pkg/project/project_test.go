package project

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

func loadProject(t *testing.T) *projectfile.Project {
	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
variables:
  - name: foo
    value: bar
  - name: bar
    value: baz
    constraints:
      platform: Linux
  - name: bar
    value: quux
    constraints:
      platform: Windows
  - name: bar
    version: "1.3"
    constraints:
      platform: Darwin
hooks:
  - name: foo
    value: bar
  - name: bar
    value: baz
    constraints:
      platform: Linux
  - name: bar
    value: quux
    constraints:
      platform: Windows
  - name: bar
    version: "1.3"
    constraints:
      platform: Darwin
commands:
  - name: foo
    value: bar
  - name: bar
    value: baz
    constraints:
      platform: Linux
  - name: bar
    value: quux
    constraints:
      platform: Windows
  - name: bar
    version: "1.3"
    constraints:
      platform: Darwin
languages:
  - name: foo
    version: "1.0"
    packages:
     - name: foo
       version: "1.0"
     - name: bar
       version: "1.1"
       constraints:
         platform: Linux
     - name: bar
       version: "1.2"
       constraints:
         platform: Windows
     - name: bar
       version: "1.3"
       constraints:
         platform: Darwin
  - name: bar
    version: "1.1"
    constraints:
      platform: Linux
  - name: bar
    version: "1.2"
    constraints:
      platform: Windows
  - name: bar
    version: "1.3"
    constraints:
      platform: Darwin
`)

	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	return project
}

func TestConstrainHooks(t *testing.T) {
	loadProject(t)
	hooks, err := Hooks()
	assert.Nil(t, err, "Run without failure")
	assert.Equal(t, 3, len(hooks), "There should be two hooks only")
}

func TestConstrainLanguages(t *testing.T) {
	loadProject(t)
	languages, err := Languages()
	assert.Nil(t, err, "Run without failure")
	assert.Equal(t, 3, len(languages), "There should be two languages only")
}

func TestConstrainPackagesOfLanguage(t *testing.T) {
	loadProject(t)
	languages, _ := Languages()
	language := languages[0]
	packages := PackagesOfLanguage(language)
	assert.Equal(t, 3, len(packages), "There should be two packages only")
}

func TestConstrainCommands(t *testing.T) {
	loadProject(t)
	commands, err := Commands()
	assert.Nil(t, err, "Run without failure")
	assert.Equal(t, 3, len(commands), "There should be two commands only")
}

func TestConstrainVariables(t *testing.T) {
	loadProject(t)
	variables, err := Variables()
	assert.Nil(t, err, "Run without failure")
	assert.Equal(t, 3, len(variables), "There should be two variables only")
}
