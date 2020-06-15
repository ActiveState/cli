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

func TestService_IncrementFrom(t *testing.T) {
	tests := []struct {
		name        string
		baseVersion string
		increment   string
		wantVersion string
		wantErr     bool
	}{
		{
			name:        "patch",
			baseVersion: "0.2.2",
			increment:   "patch",
			wantVersion: "0.2.3",
			wantErr:     false,
		},
		{
			name:        "minor",
			baseVersion: "0.2.2",
			increment:   "minor",
			wantVersion: "0.3.0",
			wantErr:     false,
		},
		{
			name:        "major",
			baseVersion: "0.2.2",
			increment:   "major",
			wantVersion: "1.0.0",
			wantErr:     false,
		},
		{
			name:        "error",
			baseVersion: "0.2.2",
			increment:   "error",
			wantVersion: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bv, err := semver.New(tt.baseVersion)
			if err != nil {
				t.Fatalf("could not parse base version string %s error = %v", tt.baseVersion, err)
			}
			nv, err := incrementFrom(bv, tt.increment)
			if (err != nil) != tt.wantErr {
				t.Errorf("incrementFrom(%s, %s) error = %v, wantErr %v", tt.baseVersion, tt.increment, err, tt.wantErr)
			}
			if nv == nil {
				return
			}
			if nv.String() != tt.wantVersion {
				t.Errorf("incrementFrom(%s, %s) == %s, want %s", tt.baseVersion, tt.increment, nv.String(), tt.wantVersion)
			}
		})
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
		name       string
		fields     fields
		wantZeroed bool
		wantErr    bool
	}{
		{
			name: "local environment",
			fields: fields{
				environment: LocalEnv,
				master:      versionSemver,
				typer:       incrementStateStore{Patch},
				branch:      "master",
			},
			wantZeroed: true,
			wantErr:    false,
		},
		{
			name: "local environment (branch)",
			fields: fields{
				environment: LocalEnv,
				master:      versionSemver,
				typer:       incrementStateStore{Patch},
				branch:      "not-master",
			},
			wantZeroed: true,
			wantErr:    false,
		},
		{
			name: "remote environment",
			fields: fields{
				environment: RemoteEnv,
				master:      versionSemver,
				typer:       incrementStateStore{Patch},
				branch:      "master",
			},
			wantZeroed: false,
			wantErr:    false,
		},
		{
			name: "remote environment - major (branch)",
			fields: fields{
				environment: RemoteEnv,
				master:      versionSemver,
				typer:       incrementStateStore{Major},
				branch:      "not-master",
			},
			wantZeroed: true,
			wantErr:    false,
		},
		{
			name: "error",
			fields: fields{
				environment: UnknownEnv,
				master:      versionSemver,
				typer:       incrementStateStore{""},
				branch:      "master",
			},
			wantZeroed: false,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Incrementation{
				env:    tt.fields.environment,
				typer:  tt.fields.typer,
				branch: tt.fields.branch,
			}
			got, err := s.Increment()
			if (err != nil) != tt.wantErr {
				t.Errorf("VersionIncrementer.IncrementVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil {
				return
			}
			gotVersion, err := semver.New(got.String())
			if err != nil {
				t.Errorf("VersionIncrementer.IncrementVersion(): Could not parse returned version %s, error = %v", got.String(), err)
			}
			isZero := gotVersion.String() == "0.0.0"
			if isZero != tt.wantZeroed {
				cond := "not"
				if tt.wantZeroed {
					cond = ""
				}
				t.Errorf("VersionIncrementer.IncrementVersion() version = %s, want %s 0.0.0", gotVersion.String(), cond)
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
		name       string
		fields     fields
		args       args
		wantZeroed bool
		wantErr    bool
	}{
		{
			name: "local environment",
			fields: fields{
				environment: LocalEnv,
				master:      versionSemver,
				provider:    incrementStateStore{Patch},
				branch:      "master",
			},
			args:       args{revision},
			wantZeroed: true,
			wantErr:    false,
		},
		{
			name: "local environment (unstable)",
			fields: fields{
				environment: LocalEnv,
				master:      versionSemver,
				provider:    incrementStateStore{Patch},
				branch:      "unstable",
			},
			args:       args{revision},
			wantZeroed: true,
			wantErr:    false,
		},
		{
			name: "local environment (branch)",
			fields: fields{
				environment: LocalEnv,
				master:      versionSemver,
				provider:    incrementStateStore{Patch},
				branch:      "not-master",
			},
			args:       args{revision},
			wantZeroed: true,
			wantErr:    false,
		},
		{
			name: "remote environment",
			fields: fields{
				environment: RemoteEnv,
				master:      versionSemver,
				provider:    incrementStateStore{Patch},
				branch:      "master",
			},
			args:       args{revision},
			wantZeroed: false,
			wantErr:    false,
		},
		{
			name: "remote environment - (unstable)",
			fields: fields{
				environment: RemoteEnv,
				master:      versionSemver,
				provider:    incrementStateStore{Major},
				branch:      "unstable",
			},
			args:       args{revision},
			wantZeroed: false,
			wantErr:    false,
		},
		{
			name: "remote environment - (branch)",
			fields: fields{
				environment: RemoteEnv,
				master:      versionSemver,
				provider:    incrementStateStore{Major},
				branch:      "not-master",
			},
			args:       args{revision},
			wantZeroed: true,
			wantErr:    false,
		},
		{
			name: "error",
			fields: fields{
				environment: UnknownEnv,
				master:      versionSemver,
				provider:    incrementStateStore{""},
				branch:      "master",
			},
			args:       args{revision},
			wantZeroed: false,
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Incrementation{
				env:    tt.fields.environment,
				typer:  tt.fields.provider,
				branch: tt.fields.branch,
			}
			got, err := s.IncrementWithRevision(tt.args.revision)
			if (err != nil) != tt.wantErr {
				t.Errorf("VersionIncrementer.IncrementVersionPreRelease() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil {
				return
			}
			gotVersion, err := semver.New(got.String())
			if err != nil {
				t.Errorf("VersionIncrementer.IncrementVersionPreRelease(): Could not parse returned version %s, error = %v", got.String(), err)
			}
			if len(gotVersion.Pre) != 1 {
				t.Errorf("VersionIncrementer.IncrementVersionPreRelease() did not return pre-release version")
			}
			if gotVersion.Pre[0].String() != fmt.Sprintf("SHA%s", revision) {
				t.Errorf("VersionIncrementer.IncrementVersionPreRelease() pre-release version = %s, want SHA%s", gotVersion.Pre[0].String(), revision)
			}
			gotVersion.Pre = nil
			isZero := gotVersion.String() == "0.0.0"
			if isZero != tt.wantZeroed {
				cond := "not"
				if tt.wantZeroed {
					cond = ""
				}
				t.Errorf("VersionIncrementer.IncrementVersionPreRelease() version = %s, want %s 0.0.0", gotVersion.String(), cond)
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
