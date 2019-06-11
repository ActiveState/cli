// +build darwin

package virtualenvironment

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
)

func TestActivateRuntimeEnvironment(t *testing.T) {
	setup(t)
	defer teardown()

	dat := strings.TrimSpace(`
project: "https://platform.activestate.com/string/string?commitID=string"
languages:
    - name: Python3`)
	yaml.Unmarshal([]byte(dat), &project)
	project.Persist()

	venv := Init()
	fail := venv.Activate()
	assert.NoError(t, fail.ToError(), "Should activate")
	assert.Empty(t, venv.artifactPaths, "Should not pull in artifacts because these are only supported on linux")
}
