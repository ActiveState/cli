// +build !linux

package virtualenvironment

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/assert"
	yaml "gopkg.in/yaml.v2"
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
	assert.Error(t, fail.ToError(), "Should not activate because Python3 is not supported on Windows yet")
	assert.Equal(t, model.FailNoEffectiveRecipe.Name, fail.Type.Name, "Should fail on unsupported language")
}
