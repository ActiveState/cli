// +build darwin

package virtualenvironment

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/pkg/projectfile"
)

func TestActivateRuntimeEnvironment(t *testing.T) {
	setup(t)
	defer teardown()

	project := &projectfile.Project{}
	dat := strings.TrimSpace(`
name: string
owner: string
languages:
    - name: Python3`)
	yaml.Unmarshal([]byte(dat), &project)
	project.Persist()

	venv := Init()
	fail := venv.Activate()
	assert.NoError(t, fail.ToError(), "Should activate")
	assert.Empty(t, venv.artifactPaths, "Should not pull in artifacts because these are only supported on linux")
}
