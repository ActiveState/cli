package captain

import (
	"reflect"
	"testing"
)

func TestUserValue_Set(t *testing.T) {
	tests := []struct {
		name      string
		flagValue string
		want      *UserValue
		wantErr   bool
	}{
		{
			"name and email",
			"John Doe <john@doe.org>",
			&UserValue{Name: "John Doe", Email: "john@doe.org"},
			false,
		},
		{
			"email only",
			"john@doe.org",
			&UserValue{Name: "john", Email: "john@doe.org"},
			false,
		},
		{
			"name only",
			"john",
			&UserValue{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UserValue{}
			if err := u.Set(tt.flagValue); (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(u, tt.want) {
				t.Fatalf("got %+v, want %+v", u, tt.want)
			}
		})
	}
}

func TestPackageValue_Set(t *testing.T) {
	tests := []struct {
		name      string
		flagValue string
		wantErr   bool
		want      *PackageValue
	}{
		{
			"namespace, name and version",
			"namespace/path:name@1.0.0",
			false,
			&PackageValue{Namespace: "namespace/path", Name: "name", Version: "1.0.0"},
		},
		{
			"namespace and name",
			"namespace/path:name",
			false,
			&PackageValue{Namespace: "namespace/path", Name: "name"},
		},
		{
			"name only",
			"name",
			false,
			&PackageValue{Name: "name"},
		},
		{
			"name and version only",
			"name@1.0.0",
			false,
			&PackageValue{Name: "name", Version: "1.0.0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PackageValue{}
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
		want      *PackageValueNSRequired
	}{
		{
			"namespace, name and version",
			"namespace/path:name@1.0.0",
			false,
			&PackageValueNSRequired{PackageValue{Namespace: "namespace/path", Name: "name", Version: "1.0.0"}},
		},
		{
			"namespace and name",
			"namespace/path:name",
			false,
			&PackageValueNSRequired{PackageValue{Namespace: "namespace/path", Name: "name"}},
		},
		{
			"name only",
			"name",
			true,
			&PackageValueNSRequired{PackageValue{}},
		},
		{
			"name and version only",
			"name@1.0.0",
			true,
			&PackageValueNSRequired{PackageValue{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PackageValueNSRequired{}
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
