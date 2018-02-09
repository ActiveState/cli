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

func TestPlatformConstraints(t *testing.T) {
	root, _ := environment.GetRootPath()
	project, err := projectfile.Parse(filepath.Join(root, "test", constants.ConfigFileName))
	assert.Nil(t, err, "There was no error parsing the config file")

	assert.True(t, platformIsConstrained("Windows10Label", project))

	osNameOverride = "linux"
	osArchitectureOverride = "amd64"
	osLibcOverride = "glibc-2.25"
	osCompilerOverride = "gcc-7"
	assert.False(t, platformIsConstrained("Linux64Label", project))
	assert.True(t, platformIsConstrained("-Linux64Label", project))
	assert.True(t, platformIsConstrained("Windows10Label", project))
	assert.False(t, platformIsConstrained("-Windows10Label", project))
	osNameOverride = ""
	osArchitectureOverride = ""
	osLibcOverride = ""
	osCompilerOverride = ""
}

func TestEnvironmentConstraints(t *testing.T) {
	os.Setenv(constants.EnvironmentEnvVarName, "dev")
	assert.False(t, environmentIsConstrained("dev"), "The current environment is in 'dev'")
	assert.False(t, environmentIsConstrained("dev,qa"), "The current environment is in 'dev,qa'")
	assert.False(t, environmentIsConstrained("qa,dev,prod"), "The current environment is in 'dev,qa,prod'")
	assert.True(t, environmentIsConstrained("qa"), "The current environment is not in 'qa'")
	assert.True(t, environmentIsConstrained("qa,devops"), "The current environment is not in 'qa,devops'")
}

func TestMatchConstraint(t *testing.T) {
	root, _ := environment.GetRootPath()
	project, err := projectfile.Parse(filepath.Join(root, "test", constants.ConfigFileName))
	assert.Nil(t, err, "There was no error parsing the config file")

	constraint := projectfile.Constraint{"Windows10Label", "dev"}
	assert.True(t, IsConstrained(constraint, project))
}
