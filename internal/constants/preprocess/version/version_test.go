package version

import (
	"errors"
	"fmt"
	"testing"

	"github.com/blang/semver"
)

type provider struct{}

func (p provider) Increment(branch string) (string, error) {
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
			s := &Service{
				branch:      tt.fields.branch,
				environment: tt.fields.environment,
				master:      tt.fields.master,
				provider:    tt.fields.provider,
			}
			got, err := s.IncrementVersion()
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.IncrementVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Service.IncrementVersion() = %v, want %v", got, tt.want)
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
			s := &Service{
				branch:      tt.fields.branch,
				environment: tt.fields.environment,
				master:      tt.fields.master,
				provider:    tt.fields.provider,
			}
			got, err := s.IncrementVersionPreRelease(tt.args.revision)
			if (err != nil) != tt.wantErr {
				t.Errorf("Service.IncrementVersionPreRelease() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Service.IncrementVersionPreRelease() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_MustIncrementVersion(t *testing.T) {
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
		name   string
		fields fields
		want   string
	}{
		{
			name: "local environment",
			fields: fields{
				branch:      patch,
				environment: localEnv,
				master:      versionSemver,
				provider:    provider{},
			},
			want: versionString,
		},
		{
			name: "remote environment - patch",
			fields: fields{
				branch:      patch,
				environment: remoteEnv,
				master:      versionSemver,
				provider:    provider{},
			},
			want: "0.2.3",
		},
		{
			name: "remote environment - minor",
			fields: fields{
				branch:      minor,
				environment: remoteEnv,
				master:      versionSemver,
				provider:    provider{},
			},
			want: "0.3.0",
		},
		{
			name: "remote environment - major",
			fields: fields{
				branch:      major,
				environment: remoteEnv,
				master:      versionSemver,
				provider:    provider{},
			},
			want: "1.0.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				branch:      tt.fields.branch,
				environment: tt.fields.environment,
				master:      tt.fields.master,
				provider:    tt.fields.provider,
			}
			if got := s.MustIncrementVersion(); got != tt.want {
				t.Errorf("Service.MustIncrementVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestService_MustIncrementVersionPreRelease(t *testing.T) {
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
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "local environment",
			fields: fields{
				branch:      patch,
				environment: localEnv,
				master:      versionSemver,
				provider:    provider{},
			},
			args: args{revision},
			want: versionString,
		},
		{
			name: "remote environment - patch",
			fields: fields{
				branch:      patch,
				environment: remoteEnv,
				master:      versionSemver,
				provider:    provider{},
			},
			args: args{revision},
			want: fmt.Sprintf("%s-%s", "0.2.3", preRelease),
		},
		{
			name: "remote environment - minor",
			fields: fields{
				branch:      minor,
				environment: remoteEnv,
				master:      versionSemver,
				provider:    provider{},
			},
			args: args{revision},
			want: fmt.Sprintf("%s-%s", "0.3.0", preRelease),
		},
		{
			name: "remote environment - major",
			fields: fields{
				branch:      major,
				environment: remoteEnv,
				master:      versionSemver,
				provider:    provider{},
			},
			args: args{revision},
			want: fmt.Sprintf("%s-%s", "1.0.0", preRelease),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Service{
				branch:      tt.fields.branch,
				environment: tt.fields.environment,
				master:      tt.fields.master,
				provider:    tt.fields.provider,
			}
			if got := s.MustIncrementVersionPreRelease(tt.args.revision); got != tt.want {
				t.Errorf("Service.MustIncrementVersionPreRelease() = %v, want %v", got, tt.want)
			}
		})
	}
}
