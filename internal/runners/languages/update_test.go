package languages

import (
	"reflect"
	"testing"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/model"
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
			"Language without version",
			args{"Python"},
			&model.Language{"Python", ""},
			false,
		},
		{
			"Language with version",
			args{"Python@2"},
			&model.Language{"Python", "2"},
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
		latestVersion latestVersionFunc
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
				func(name string) ([]string, *failures.Failure) { return []string{"2.0", "3.5", "4.0"}, nil },
				func(name string) (string, error) { return "", nil },
			},
			"3.5",
			false,
		},
		{
			"Version latest",
			args{
				&model.Language{"Python", ""},
				func(name string) ([]string, *failures.Failure) { return []string{"2.0", "3.5", "2.7"}, nil },
				func(name string) (string, error) { return "3.5", nil },
			},
			"3.5",
			false,
		},
		{
			"Version latest not available",
			args{
				&model.Language{"Python", ""},
				func(name string) ([]string, *failures.Failure) { return []string{"2.0", "3.5", "4.0"}, nil },
				func(name string) (string, error) { return "5.0", nil },
			},
			"Irrelevant, should fail",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ensureVersionTestable(tt.args.language, tt.args.fetchVersions, tt.args.latestVersion)

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

func Test_latestVersionTestable(t *testing.T) {
	type args struct {
		name          string
		fetchVersions fetchVersionsFunc
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			"Latest Version",
			args{
				"python",
				func(name string) ([]string, *failures.Failure) { return []string{"2.0", "3.5", "4.0", "1.0"}, nil },
			},
			"4.0",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := latestVersionTestable(tt.args.name, tt.args.fetchVersions)
			if (err != nil) != tt.wantErr {
				t.Errorf("latestVersionTestable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("latestVersionTestable() got = %v, want %v", got, tt.want)
			}
		})
	}
}
