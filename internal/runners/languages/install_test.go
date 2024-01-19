package languages

import (
	"github.com/ActiveState/cli/pkg/platform/model"
	"reflect"
	"testing"
)

func Test_parseLanguage(t *testing.T) {
	type args struct {
		langName string
	}
	tests := []struct {
		name    string
		args    args
		want    *model.Language
		wantErr bool
	}{
		{
			"Language with version",
			args{"Python@2"},
			&model.Language{Name: "Python", Version: "2"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLanguage(tt.args.langName)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseLanguage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseLanguage() got = %v, want %v", got, tt.want)
			}
		})
	}
}

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
				&model.Language{Name: "Python", Version: "3.5"},
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
