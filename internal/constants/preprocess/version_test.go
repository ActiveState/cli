package preprocess

import (
	"errors"
	"fmt"
	"testing"

	"github.com/blang/semver"
)

type provider struct{}

func (p provider) IncrementType(branch string) (string, error) {
	switch branch {
	case patch:
		return patch, nil
	case minor:
		return minor, nil
	case major:
		return major, nil
	default:
		return "", errors.New("error")
	}
}

func TestService_IncrementVersion(t *testing.T) {
	versionString := "0.2.2"
	versionSemver, err := semver.New("0.2.2")
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		branch      string
		environment int
		master      *semver.Version
		provider    IncrementProvider
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
				branch:      patch,
				environment: localEnv,
				master:      versionSemver,
				provider:    provider{},
			},
			want:    versionString,
			wantErr: false,
		},
		{
			name: "remote environment - patch",
			fields: fields{
				branch:      patch,
				environment: remoteEnv,
				master:      versionSemver,
				provider:    provider{},
			},
			want:    "0.2.3",
			wantErr: false,
		},
		{
			name: "remote environment - minor",
			fields: fields{
				branch:      minor,
				environment: remoteEnv,
				master:      versionSemver,
				provider:    provider{},
			},
			want:    "0.3.0",
			wantErr: false,
		},
		{
			name: "remote environment - major",
			fields: fields{
				branch:      major,
				environment: remoteEnv,
				master:      versionSemver,
				provider:    provider{},
			},
			want:    "1.0.0",
			wantErr: false,
		},
		{
			name: "error",
			fields: fields{
				branch:      "",
				environment: unknownEnv,
				master:      versionSemver,
				provider:    provider{},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &VersionIncrementer{
				branch:      tt.fields.branch,
				environment: tt.fields.environment,
				master:      tt.fields.master,
				provider:    tt.fields.provider,
			}
			got, err := s.IncrementVersion()
			if (err != nil) != tt.wantErr {
				t.Errorf("VersionIncrementer.IncrementVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("VersionIncrementer.IncrementVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_IncrementVersionPreRelease(t *testing.T) {
	versionString := "0.2.2-1a2b3c4d"
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
		branch      string
		environment int
		master      *semver.Version
		provider    IncrementProvider
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
				branch:      patch,
				environment: localEnv,
				master:      versionSemver,
				provider:    provider{},
			},
			args:    args{revision},
			want:    versionString,
			wantErr: false,
		},
		{
			name: "remote environment - patch",
			fields: fields{
				branch:      patch,
				environment: remoteEnv,
				master:      versionSemver,
				provider:    provider{},
			},
			args:    args{revision},
			want:    fmt.Sprintf("%s-%s", "0.2.3", preRelease),
			wantErr: false,
		},
		{
			name: "remote environment - minor",
			fields: fields{
				branch:      minor,
				environment: remoteEnv,
				master:      versionSemver,
				provider:    provider{},
			},
			args:    args{revision},
			want:    fmt.Sprintf("%s-%s", "0.3.0", preRelease),
			wantErr: false,
		},
		{
			name: "remote environment - major",
			fields: fields{
				branch:      major,
				environment: remoteEnv,
				master:      versionSemver,
				provider:    provider{},
			},
			args:    args{revision},
			want:    fmt.Sprintf("%s-%s", "1.0.0", preRelease),
			wantErr: false,
		},
		{
			name: "error",
			fields: fields{
				branch:      "",
				environment: unknownEnv,
				master:      versionSemver,
				provider:    provider{},
			},
			args:    args{revision},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &VersionIncrementer{
				branch:      tt.fields.branch,
				environment: tt.fields.environment,
				master:      tt.fields.master,
				provider:    tt.fields.provider,
			}
			got, err := s.IncrementVersionPreRelease(tt.args.revision)
			if (err != nil) != tt.wantErr {
				t.Errorf("VersionIncrementer.IncrementVersionPreRelease() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("VersionIncrementer.IncrementVersionPreRelease() = %v, want %v", got, tt.want)
			}
		})
	}
}
