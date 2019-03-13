package scripts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
		err := Command.Execute()
		assert.NoError(t, err, "Executed without error")
		assert.NoError(t, failures.Handled(), "No failure occurred")
	}

	{
		str, err := osutil.CaptureStdout(func() {
			Command.Execute()
		})
		assert.NoError(t, err, "Executed without error")
		assert.Equal(t, " * run\n", str, "Outputs don't match")
	}

}
