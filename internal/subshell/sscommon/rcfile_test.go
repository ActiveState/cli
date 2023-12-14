package sscommon

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
)

func fakeContents(before, contents, after string) string {
	var blocks []string
	if before != "" {
		blocks = append(blocks, before)
	}
	if contents != "" {
		blocks = append(
			blocks,
			fmt.Sprintf("# %s", constants.RCAppendDefaultStartLine),
			contents,
			fmt.Sprintf("# %s", constants.RCAppendDefaultStopLine),
		)
	}
	if after != "" {
		blocks = append(blocks, after)
	}

	return strings.Join(blocks, fileutils.LineEnd)
}

func fakeFileWithContents(before, contents, after string) string {
	f := fileutils.TempFileUnsafe()
	defer f.Close()
	f.WriteString(fakeContents(before, contents, after))
	return f.Name()
}

func TestWriteRcFile(t *testing.T) {
	type args struct {
		rcTemplateName string
		path           string
		env            map[string]string
	}

	fish := fmt.Sprintf(
		`set -xg PATH "foo:$PATH"
if test ! -z "$%s"; test -f "$%s/%s"
  echo "State Tool is operating on project $%s, located at $%s"
end`,
		constants.ActivatedStateEnvVarName,
		constants.ActivatedStateEnvVarName,
		constants.ConfigFileName,
		constants.ActivatedStateNamespaceEnvVarName,
		constants.ActivatedStateEnvVarName)
	if runtime.GOOS == "windows" {
		fish = strings.ReplaceAll(fish, "\n", "\r\n")
	}

	tests := []struct {
		name         string
		args         args
		want         error
		wantContents string
	}{
		{
			"Write RC to empty file",
			args{
				"fishrc_append.fish",
				fakeFileWithContents("", "", ""),
				map[string]string{
					"PATH": "foo",
				},
			},
			nil,
			fakeContents("", fish, ""),
		},
		{
			"Write RC update",
			args{
				"fishrc_append.fish",
				fakeFileWithContents("before", "SOMETHING ELSE", "after"),
				map[string]string{
					"PATH": "foo",
				},
			},
			nil,
			fakeContents(strings.Join([]string{"before", "after"}, fileutils.LineEnd), fish, ""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WriteRcFile(tt.args.rcTemplateName, tt.args.path, DefaultID, tt.args.env); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WriteRcFile() = %v, want %v", got, tt.want)
			}
			if !fileutils.FileExists(tt.args.path) {
				t.Errorf("File does not exist: %s", tt.args.path)
			}
			data := fileutils.ReadFileUnsafe(tt.args.path)
			contents := strings.TrimSpace(string(data))
			if contents != tt.wantContents {
				t.Errorf("File contents don't match, got = '%s' want ='%s'", contents, tt.wantContents)
			}
		})
	}
}
