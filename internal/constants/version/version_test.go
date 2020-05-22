package version

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
	case Patch, Minor, Major:
		return p.increment, nil
	case Zeroed:
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
		environment Env
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
				environment: LocalEnv,
				master:      versionSemver,
				typer:       incrementStateStore{Patch},
				branch:      "master",
			},
			want:    "0.0.0",
			wantErr: false,
		},
		{
			name: "local environment (branch)",
			fields: fields{
				environment: LocalEnv,
				master:      versionSemver,
				typer:       incrementStateStore{Patch},
				branch:      "not-master",
			},
			want:    "0.0.0",
			wantErr: false,
		},
		{
			name: "remote environment - patch",
			fields: fields{
				environment: RemoteEnv,
				master:      versionSemver,
				typer:       incrementStateStore{Patch},
				branch:      "master",
			},
			want:    "0.2.3",
			wantErr: false,
		},
		{
			name: "remote environment - minor",
			fields: fields{
				environment: RemoteEnv,
				master:      versionSemver,
				typer:       incrementStateStore{Minor},
				branch:      "master",
			},
			want:    "0.3.0",
			wantErr: false,
		},
		{
			name: "remote environment - major",
			fields: fields{
				environment: RemoteEnv,
				master:      versionSemver,
				typer:       incrementStateStore{Major},
				branch:      "master",
			},
			want:    "1.0.0",
			wantErr: false,
		},
		{
			name: "remote environment - major (branch)",
			fields: fields{
				environment: RemoteEnv,
				master:      versionSemver,
				typer:       incrementStateStore{Major},
				branch:      "not-master",
			},
			want:    "0.0.0",
			wantErr: false,
		},
		{
			name: "remote environment - patch (branch)",
			fields: fields{
				environment: RemoteEnv,
				master:      versionSemver,
				typer:       incrementStateStore{Patch},
				branch:      "not-master",
			},
			want:    "0.0.0",
			wantErr: false,
		},
		{
			name: "error",
			fields: fields{
				environment: UnknownEnv,
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
			s := &Incrementation{
				env:    tt.fields.environment,
				master: tt.fields.master,
				typer:  tt.fields.typer,
				branch: tt.fields.branch,
			}
			got, err := s.Increment()
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
		environment Env
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
				environment: LocalEnv,
				master:      versionSemver,
				provider:    incrementStateStore{Patch},
				branch:      "master",
			},
			args:    args{revision},
			want:    fmt.Sprintf("%s-SHA%s", "0.0.0", preRelease),
			wantErr: false,
		},
		{
			name: "local environment (unstable)",
			fields: fields{
				environment: LocalEnv,
				master:      versionSemver,
				provider:    incrementStateStore{Patch},
				branch:      "unstable",
			},
			args:    args{revision},
			want:    fmt.Sprintf("%s-SHA%s", "0.0.0", preRelease),
			wantErr: false,
		},
		{
			name: "local environment (branch)",
			fields: fields{
				environment: LocalEnv,
				master:      versionSemver,
				provider:    incrementStateStore{Patch},
				branch:      "not-master",
			},
			args:    args{revision},
			want:    fmt.Sprintf("%s-SHA%s", "0.0.0", preRelease),
			wantErr: false,
		},
		{
			name: "remote environment - patch",
			fields: fields{
				environment: RemoteEnv,
				master:      versionSemver,
				provider:    incrementStateStore{Patch},
				branch:      "master",
			},
			args:    args{revision},
			want:    fmt.Sprintf("%s-SHA%s", "0.2.3", preRelease),
			wantErr: false,
		},
		{
			name: "remote environment - minor",
			fields: fields{
				environment: RemoteEnv,
				master:      versionSemver,
				provider:    incrementStateStore{Minor},
				branch:      "master",
			},
			args:    args{revision},
			want:    fmt.Sprintf("%s-SHA%s", "0.3.0", preRelease),
			wantErr: false,
		},
		{
			name: "remote environment - major",
			fields: fields{
				environment: RemoteEnv,
				master:      versionSemver,
				provider:    incrementStateStore{Major},
				branch:      "master",
			},
			args:    args{revision},
			want:    fmt.Sprintf("%s-SHA%s", "1.0.0", preRelease),
			wantErr: false,
		},
		{
			name: "remote environment - major (unstable)",
			fields: fields{
				environment: RemoteEnv,
				master:      versionSemver,
				provider:    incrementStateStore{Major},
				branch:      "unstable",
			},
			args:    args{revision},
			want:    fmt.Sprintf("%s-SHA%s", "1.0.0", preRelease),
			wantErr: false,
		},
		{
			name: "remote environment - major (branch)",
			fields: fields{
				environment: RemoteEnv,
				master:      versionSemver,
				provider:    incrementStateStore{Major},
				branch:      "not-master",
			},
			args:    args{revision},
			want:    fmt.Sprintf("%s-SHA%s", "0.0.0", preRelease),
			wantErr: false,
		},
		{
			name: "remote environment - patch (branch)",
			fields: fields{
				environment: RemoteEnv,
				master:      versionSemver,
				provider:    incrementStateStore{Patch},
				branch:      "not-master",
			},
			args:    args{revision},
			want:    fmt.Sprintf("%s-SHA%s", "0.0.0", preRelease),
			wantErr: false,
		},
		{
			name: "error",
			fields: fields{
				environment: UnknownEnv,
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
			s := &Incrementation{
				env:    tt.fields.environment,
				master: tt.fields.master,
				typer:  tt.fields.provider,
				branch: tt.fields.branch,
			}
			got, err := s.IncrementWithRevision(tt.args.revision)
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

func TestNumberIsProduction(t *testing.T) {
	tests := []struct {
		num  string
		want bool
	}{
		{"0.0.0", false},
		{"0.0.1", true},
		{"0.1.0", true},
		{"1.0.0", true},
		{"1.1.0", true},
		{"0.1.1", true},
		{"junk", false},
	}

	for _, tt := range tests {
		got := NumberIsProduction(tt.num)
		if got != tt.want {
			t.Errorf("%q: got %v, want %v", tt.num, got, tt.want)
		}
	}
}
