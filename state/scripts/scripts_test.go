package scripts

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"

	"github.com/ActiveState/cli/internal/failures"
	osutil "github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func setup(t *testing.T) {
	Flags.Output = new(string)
}

func TestExecute(t *testing.T) {
	setup(t)

	project := &projectfile.Project{}
	contents := strings.TrimSpace(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
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
		assert.NotEmpty(t, str, "Outputs don't match")
	}

}

func TestScriptsTable(t *testing.T) {
	setup(t)

	hdrs := []string{"Name", "Description"}
	rows := [][]string{
		{"name0", "desc0"},
		{"name1", "desc1"},
		{"name2", "desc2"},
	}

	tests := []struct {
		name     string
		ss       []*project.Script
		wantHdrs []string
		wantRows [][]string
	}{
		{
			"basic",
			[]*project.Script{
				newScript(t, rows[0][0], rows[0][1], ""),
				newScript(t, rows[1][0], rows[1][1], ""),
				newScript(t, rows[2][0], rows[2][1], ""),
			},
			hdrs, rows,
		},
		{
			"basic-reversed",
			[]*project.Script{
				newScript(t, rows[2][0], rows[2][1], ""),
				newScript(t, rows[1][0], rows[1][1], ""),
				newScript(t, rows[0][0], rows[0][1], ""),
			},
			hdrs,
			[][]string{rows[2], rows[1], rows[0]},
		},
	}

	for _, tt := range tests {
		gotHdrs, gotRows := scriptsTable(tt.ss)
		assert.Equalf(t, tt.wantHdrs, gotHdrs, "headers mismatch for %q", tt.name)
		assert.Equalf(t, tt.wantRows, gotRows, "rows mismatch for %q", tt.name)
	}
}

func newScript(t *testing.T, name, desc, val string) *project.Script {
	pjFile := projectfile.Project{}
	contents := strings.TrimSpace(fmt.Sprintf(`
project: "https://platform.activestate.com/ActiveState/project?commitID=00010001-0001-0001-0001-000100010001"
scripts:
  - name: %s
    description: %s
    value: %s
`, name, desc, val))

	err := yaml.Unmarshal([]byte(contents), &pjFile)
	assert.Nil(t, err, "Unmarshalled YAML")

	prj, fail := project.New(&pjFile)
	assert.Nil(t, fail, "no failure should occur when loading project")
	return prj.ScriptByName(name)
}
