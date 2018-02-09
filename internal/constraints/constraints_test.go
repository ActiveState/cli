package constraints

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	"github.com/ActiveState/ActiveState-CLI/pkg/projectfile"
	"github.com/stretchr/testify/assert"
)

func TestMatchPlatform(t *testing.T) {
	root, _ := environment.GetRootPath()
	project, err := projectfile.Parse(filepath.Join(root, "test", constants.ConfigFileName))
	assert.Nil(t, err, "There was no error parsing the config file")

	assert.False(t, MatchesPlatform("Windows10Label", project))

	osNameOverride = "linux"
	osArchitectureOverride = "amd64"
	osLibcOverride = "glibc-2.25"
	osCompilerOverride = "gcc-7"
	assert.True(t, MatchesPlatform("Linux64Label", project))
	assert.False(t, MatchesPlatform("-Linux64Label", project))
	assert.False(t, MatchesPlatform("Windows10Label", project))
	assert.True(t, MatchesPlatform("-Windows10Label", project))
}

func TestMatchEnvironment(t *testing.T) {
	os.Setenv(constants.EnvironmentEnvVarName, "dev")
	assert.True(t, MatchesEnvironment("dev"), "The current environment is in 'dev'")
	assert.True(t, MatchesEnvironment("dev,qa"), "The current environment is in 'dev,qa'")
	assert.True(t, MatchesEnvironment("qa,dev,prod"), "The current environment is in 'dev,qa,prod'")
	assert.False(t, MatchesEnvironment("qa"), "The current environment is not in 'qa'")
	assert.False(t, MatchesEnvironment("qa,devops"), "The current environment is not in 'qa,devops'")
}
