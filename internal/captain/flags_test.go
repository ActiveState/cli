package captain

import (
	"reflect"
	"testing"
)

func TestUserFlag_Set(t *testing.T) {
	tests := []struct {
		name      string
		flagValue string
		want      *UserFlag
		wantErr   bool
	}{
		{
			"name and email",
			"John Doe <john@doe.org>",
			&UserFlag{Name: "John Doe", Email: "john@doe.org"},
			false,
		},
		{
			"email only",
			"john@doe.org",
			&UserFlag{Name: "john", Email: "john@doe.org"},
			false,
		},
		{
			"name only",
			"john",
			&UserFlag{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UserFlag{}
			if err := u.Set(tt.flagValue); (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(u, tt.want) {
				t.Fatalf("got %+v, want %+v", u, tt.want)
			}
		})
	}
}

func TestPackageFlag_Set(t *testing.T) {
	tests := []struct {
		name      string
		flagValue string
		wantErr   bool
		want      *PackageFlag
	}{
		{
			"namespace, name and version",
			"namespace/path/name@1.0.0",
			false,
			&PackageFlag{Namespace: "namespace/path", Name: "name", Version: "1.0.0"},
		},
		{
			"namespace and name",
			"namespace/path/name",
			false,
			&PackageFlag{Namespace: "namespace/path", Name: "name"},
		},
		{
			"name only",
			"name",
			false,
			&PackageFlag{Name: "name"},
		},
		{
			"name and version only",
			"name@1.0.0",
			false,
			&PackageFlag{Name: "name", Version: "1.0.0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PackageFlag{}
			if err := p.Set(tt.flagValue); (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(p, tt.want) {
				t.Fatalf("got %+v, want %+v", p, tt.want)
			}
		})
	}
}

func TestPackageFlagNSRequired_Set(t *testing.T) {
	tests := []struct {
		name      string
		flagValue string
		wantErr   bool
		want      *PackageFlagNSRequired
	}{
		{
			"namespace, name and version",
			"namespace/path/name@1.0.0",
			false,
			&PackageFlagNSRequired{PackageFlag{Namespace: "namespace/path", Name: "name", Version: "1.0.0"}},
		},
		{
			"namespace and name",
			"namespace/path/name",
			false,
			&PackageFlagNSRequired{PackageFlag{Namespace: "namespace/path", Name: "name"}},
		},
		{
			"name only",
			"name",
			true,
			&PackageFlagNSRequired{PackageFlag{}},
		},
		{
			"name and version only",
			"name@1.0.0",
			true,
			&PackageFlagNSRequired{PackageFlag{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PackageFlagNSRequired{}
			if err := p.Set(tt.flagValue); (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(p, tt.want) {
				t.Fatalf("got %#v, want %#v", p, tt.want)
			}
		})
	}
}
