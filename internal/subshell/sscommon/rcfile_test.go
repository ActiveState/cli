package sscommon

import (
	"reflect"
	"testing"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
)

func TestWriteRcFile(t *testing.T) {
	type args struct {
		rcTemplateName string
		path           string
		env            map[string]string
	}
	tests := []struct {
		name string
		args args
		want *failures.Failure
	}{
		{
			"Write RC",
			args{
				"fishrc_append.fish",
				fileutils.TempFileUnsafe().Name(),
				map[string]string{
					"PATH": "foo",
				},
			},
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WriteRcFile(tt.args.rcTemplateName, tt.args.path, tt.args.env); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WriteRcFile() = %v, want %v", got, tt.want)
			}
			if ! fileutils.FileExists(tt.args.path) {
				t.Errorf("File does not exist: %s", tt.args.path)
			}
			if len(fileutils.ReadFileUnsafe(tt.args.path)) == 0 {
				t.Errorf("File is empty: %s", tt.args.path)
			}
		})
	}
}
