// +build linux

package virtualenvironment

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/sysinfo"
)

func init() {
	// Only linux is supported for now, so force it so we can run this test on mac
	// If we want to skip this on mac it should be skipped through build tags, in
	// which case this tweak is meaningless and only a convenience for when testing manually
	model.OS = sysinfo.Linux
	OS = "linux"
}

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
	assert.NotEmpty(t, venv.artifactPaths, "Pulled in artifacts")

	for _, path := range venv.artifactPaths {
		assert.Contains(t, venv.GetEnv()["PATH"], path, "Artifact path is added to PATH")
	}
}
