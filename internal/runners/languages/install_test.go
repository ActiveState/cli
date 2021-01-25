package languages

import (
	"testing"

	"github.com/ActiveState/cli/pkg/platform/model"
)

func Test_ensureVersionTestable(t *testing.T) {
	type args struct {
		language      *model.Language
		fetchVersions fetchVersionsFunc
	}
	tests := []struct {
		name        string
		args        args
		wantVersion string
		wantErr     bool
	}{
		{
			"Version matches",
			args{
				&model.Language{"Python", "3.5"},
				func(name string) ([]string, error) { return []string{"2.0", "3.5", "4.0"}, nil },
			},
			"3.5",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ensureVersionTestable(tt.args.language, tt.args.fetchVersions)

			if (err != nil) != tt.wantErr {
				t.Errorf("ensureVersionTestable() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				return
			}

			if tt.args.language.Version != tt.wantVersion {
				t.Errorf("ensureVersionTestable() version = %v, wantVersion %v", tt.args.language.Version, tt.wantVersion)
			}
		})
	}
}
