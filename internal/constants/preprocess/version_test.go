package preprocess

import (
	"errors"
	"fmt"
	"testing"

	"github.com/blang/semver"
)

type incrementStateStore struct {
	increment string
}

func (p incrementStateStore) IncrementType() (string, error) {
	switch p.increment {
	case patch, minor, major:
		return p.increment, nil
	case zeroed:
		return "", errors.New("should never return zeroed")
	default:
		return "", errors.New("unknown increment type name")
	}
}

func TestService_IncrementVersion(t *testing.T) {
	versionSemver, err := semver.New("0.2.2")
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		environment int
		master      *semver.Version
		typer       IncrementTyper
		branch      string
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "local environment",
			fields: fields{
				environment: localEnv,
				master:      versionSemver,
				typer:       incrementStateStore{patch},
				branch:      "master",
			},
			want:    "0.0.0",
			wantErr: false,
		},
		{
			name: "local environment (branch)",
			fields: fields{
				environment: localEnv,
				master:      versionSemver,
				typer:       incrementStateStore{patch},
				branch:      "not-master",
			},
			want:    "0.0.0",
			wantErr: false,
		},
		{
			name: "remote environment - patch",
			fields: fields{
				environment: remoteEnv,
				master:      versionSemver,
				typer:       incrementStateStore{patch},
				branch:      "master",
			},
			want:    "0.2.3",
			wantErr: false,
		},
		{
			name: "remote environment - minor",
			fields: fields{
				environment: remoteEnv,
				master:      versionSemver,
				typer:       incrementStateStore{minor},
				branch:      "master",
			},
			want:    "0.3.0",
			wantErr: false,
		},
		{
			name: "remote environment - major",
			fields: fields{
				environment: remoteEnv,
				master:      versionSemver,
				typer:       incrementStateStore{major},
				branch:      "master",
			},
			want:    "1.0.0",
			wantErr: false,
		},
		{
			name: "remote environment - major (branch)",
			fields: fields{
				environment: remoteEnv,
				master:      versionSemver,
				typer:       incrementStateStore{major},
				branch:      "not-master",
			},
			want:    "0.0.0",
			wantErr: false,
		},
		{
			name: "remote environment - patch (branch)",
			fields: fields{
				environment: remoteEnv,
				master:      versionSemver,
				typer:       incrementStateStore{patch},
				branch:      "not-master",
			},
			want:    "0.0.0",
			wantErr: false,
		},
		{
			name: "error",
			fields: fields{
				environment: unknownEnv,
				master:      versionSemver,
				typer:       incrementStateStore{""},
				branch:      "master",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &VersionIncrementer{
				environment: tt.fields.environment,
				master:      tt.fields.master,
				typer:       tt.fields.typer,
				branch:      tt.fields.branch,
			}
			got, err := s.IncrementVersion()
			if (err != nil) != tt.wantErr {
				t.Errorf("VersionIncrementer.IncrementVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			var gotString string
			if got != nil {
				gotString = got.String()
			}
			if gotString != tt.want {
				t.Errorf("VersionIncrementer.IncrementVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_IncrementVersionPreRelease(t *testing.T) {
	versionSemver, err := semver.New("0.2.2")
	if err != nil {
		t.Fatal(err)
	}

	revision := "1a2b3c4d"
	preRelease, err := semver.NewPRVersion(revision)
	if err != nil {
		t.Fatal(err)
	}
	versionSemver.Pre = []semver.PRVersion{preRelease}

	type fields struct {
		environment int
		master      *semver.Version
		provider    IncrementTyper
		branch      string
	}
	type args struct {
		revision string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "local environment",
			fields: fields{
				environment: localEnv,
				master:      versionSemver,
				provider:    incrementStateStore{patch},
				branch:      "master",
			},
			args:    args{revision},
			want:    fmt.Sprintf("%s-SHA%s", "0.0.0", preRelease),
			wantErr: false,
		},
		{
			name: "local environment (branch)",
			fields: fields{
				environment: localEnv,
				master:      versionSemver,
				provider:    incrementStateStore{patch},
				branch:      "not-master",
			},
			args:    args{revision},
			want:    fmt.Sprintf("%s-SHA%s", "0.0.0", preRelease),
			wantErr: false,
		},
		{
			name: "remote environment - patch",
			fields: fields{
				environment: remoteEnv,
				master:      versionSemver,
				provider:    incrementStateStore{patch},
				branch:      "master",
			},
			args:    args{revision},
			want:    fmt.Sprintf("%s-SHA%s", "0.2.3", preRelease),
			wantErr: false,
		},
		{
			name: "remote environment - minor",
			fields: fields{
				environment: remoteEnv,
				master:      versionSemver,
				provider:    incrementStateStore{minor},
				branch:      "master",
			},
			args:    args{revision},
			want:    fmt.Sprintf("%s-SHA%s", "0.3.0", preRelease),
			wantErr: false,
		},
		{
			name: "remote environment - major",
			fields: fields{
				environment: remoteEnv,
				master:      versionSemver,
				provider:    incrementStateStore{major},
				branch:      "master",
			},
			args:    args{revision},
			want:    fmt.Sprintf("%s-SHA%s", "1.0.0", preRelease),
			wantErr: false,
		},
		{
			name: "remote environment - major (branch)",
			fields: fields{
				environment: remoteEnv,
				master:      versionSemver,
				provider:    incrementStateStore{major},
				branch:      "not-master",
			},
			args:    args{revision},
			want:    fmt.Sprintf("%s-SHA%s", "0.0.0", preRelease),
			wantErr: false,
		},
		{
			name: "remote environment - patch (branch)",
			fields: fields{
				environment: remoteEnv,
				master:      versionSemver,
				provider:    incrementStateStore{patch},
				branch:      "not-master",
			},
			args:    args{revision},
			want:    fmt.Sprintf("%s-SHA%s", "0.0.0", preRelease),
			wantErr: false,
		},
		{
			name: "error",
			fields: fields{
				environment: unknownEnv,
				master:      versionSemver,
				provider:    incrementStateStore{""},
				branch:      "master",
			},
			args:    args{revision},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &VersionIncrementer{
				environment: tt.fields.environment,
				master:      tt.fields.master,
				typer:       tt.fields.provider,
				branch:      tt.fields.branch,
			}
			got, err := s.IncrementVersionRevision(tt.args.revision)
			if (err != nil) != tt.wantErr {
				t.Errorf("VersionIncrementer.IncrementVersionPreRelease() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			var gotString string
			if got != nil {
				gotString = got.String()
			}
			if gotString != tt.want {
				t.Errorf("VersionIncrementer.IncrementVersionPreRelease() = %v, want %v", got, tt.want)
			}
		})
	}
}
