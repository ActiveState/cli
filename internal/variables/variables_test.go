package variables

import (
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

func TestExpandFromProject(t *testing.T) {
	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
platforms:
  - name: Linux
    os: linux
  - name: Windows
    os: windows
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
hooks:
  - name: pre
    value: echo 'Hello $variables.foo!'
  - name: post
    value: echo 'Hello $variables.bar!'
commands:
  - name: test
    value: make test
  - name: recursive
    value: $commands.recursive
  `)

	err := yaml.Unmarshal([]byte(contents), project)
	assert.Nil(t, err, "Unmarshalled YAML")
	project.Persist()

	assert.Equal(t, runtime.GOOS, ExpandFromProject("$platform.os", project), "Expanded platform variable")
	assert.Equal(t, "echo 'Hello bar!'", ExpandFromProject("$hooks.pre", project), "Expanded simple variable")
	if runtime.GOOS == "linux" {
		assert.Equal(t, "echo 'Hello baz!'", ExpandFromProject("$hooks.post", project), "Expanded platform-specific variable")
	} else if runtime.GOOS == "windows" {
		assert.Equal(t, "echo 'Hello quux!'", ExpandFromProject("$hooks.post", project), "Expanded platform-specific variable")
	}
	assert.Equal(t, "$ make test", ExpandFromProject("$ $commands.test", project), "Expanded simple command")

	assert.Equal(t, "oops: ", ExpandFromProject("oops: $commands.recursive", project), "Handled infinite recursion")
	assert.Equal(t, "", ExpandFromProject("$variables.unknown", project), "Handled undefined variables")
}
