// +build linux

package virtualenvironment

import (
	"strings"
	"testing"

	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, fail.ToError(), "Should activate")
}
