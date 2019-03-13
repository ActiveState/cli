package scripts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/failures"
	osutil "github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func TestExecute(t *testing.T) {
	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
scripts:
  - name: run
    value: whatever
  `)
	{
		err := yaml.Unmarshal([]byte(contents), project)
		assert.Nil(t, err, "Unmarshalled YAML")
		project.Persist()
	}

	{
		var otherErr error
		str, err := osutil.CaptureStdout(func() {
			otherErr = Command.Execute()
		})
		assert.NoError(t, err, "Error capturing Execute output")
		require.NoError(t, otherErr, "Should Executed without error")
		assert.NoError(t, failures.Handled(), "No failure should occurr")
		assert.Equal(t, " * run\n", str, "Outputs don't match")
	}

}
