package exeutils

import (
	"reflect"
	"sort"
	"testing"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/testhelpers/osutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_uniqueExes(t *testing.T) {
	tests := []struct {
		name    string
		bins    []string
		pathext string
		want    []string
	}{
		{
			"Returns same bins",
			[]string{"path1/a", "path2/b", "path3/c"},
			"",
			[]string{"path1/a", "path2/b", "path3/c"},
		},
		{
			"Returns exe prioritized",
			[]string{"path1/a.exe", "path2/a.cmd", "path3/c"},
			".exe;.cmd",
			[]string{"path1/a.exe", "path3/c"},
		},
		{
			"Returns cmd prioritized by PATH",
			[]string{"path1/a.exe", "path2/a.cmd", "path2/c"},
			".cmd;.exe",
			[]string{"path1/a.exe", "path2/c"},
		},
		{
			"Returns cmd prioritized by PATHEXT",
			[]string{"path1/a.exe", "path1/a.cmd", "path2/c"},
			".cmd;.exe",
			[]string{"path1/a.cmd", "path2/c"},
		},
		{
			"PATHEXT can be empty",
			[]string{"path1/a", "path2/b", "path3/c"},
			"",
			[]string{"path1/a", "path2/b", "path3/c"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UniqueExes(tt.bins, tt.pathext)
			if err != nil {
				t.Errorf("uniqueExes error: %v", err)
				t.FailNow()
			}
			sort.Strings(got)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("uniqueExes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecuteAndPipeStd(t *testing.T) {
	out, err := osutil.CaptureStdout(func() {
		logging.SetLevel(logging.NOTHING)
		defer logging.SetLevel(logging.NORMAL)
		ExecuteAndPipeStd("printenv", []string{"FOO"}, []string{"FOO=--out--"})
	})
	require.NoError(t, err)
	assert.Equal(t, "--out--\n", out, "captures output")
}
