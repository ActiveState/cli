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
	}{
		{
			"name and email",
			"John Doe <john@doe.org>",
			&UserFlag{Name: "John Doe", Email: "john@doe.org"},
		},
		{
			"email only",
			"john@doe.org",
			&UserFlag{Name: "", Email: "john@doe.org"},
		},
		{
			"name only",
			"john",
			&UserFlag{Name: "john", Email: ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UserFlag{}
			if err := u.Set(tt.flagValue); err != nil {
				t.Errorf("Set() error = %v", err)
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
			true,
			&PackageFlag{},
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
