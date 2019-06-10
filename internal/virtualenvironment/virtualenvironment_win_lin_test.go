// +build !darwin

package virtualenvironment

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/ActiveState/sysinfo"
)

func init() {
	// Only linux is supported for now, so force it so we can run this test on mac
	// If we want to skip this on mac it should be skipped through build tags, in
	// which case this tweak is meaningless and only a convenience for when testing manually
	if runtime.GOOS == "darwin" {
		model.OS = sysinfo.Linux
		OS = "linux"
	}
}

func TestActivateRuntimeEnvironment(t *testing.T) {
	setup(t)
	defer teardown()

	os.Unsetenv(constants.DisableRuntime)

	project := projectfile.Project{}
	dat := strings.TrimSpace(`
project: "https://platform.activestate.com/string/string/d7ebc72"
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

	env := venv.GetEnv()
	for k := range env {
		assert.NotEmpty(t, k, "Does not return any empty env keys")
	}
}

func TestSkipActivateRuntimeEnvironment(t *testing.T) {
	setup(t)
	defer teardown()

	os.Setenv(constants.DisableRuntime, "true")
	defer os.Unsetenv(constants.DisableRuntime)

	project := projectfile.Project{}
	dat := strings.TrimSpace(`
project: "https://platform.activestate.com/string/string/d7ebc72"
languages:
    - name: Python3`)
	yaml.Unmarshal([]byte(dat), &project)
	project.Persist()

	venv := Init()
	fail := venv.Activate()
	require.NoError(t, fail.ToError(), "Should activate")
	assert.Empty(t, venv.artifactPaths, "Did not Pull in artifacts")
}
