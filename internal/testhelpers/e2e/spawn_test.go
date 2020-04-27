package e2e

import (
	"reflect"
	"testing"
)

func TestAppendEnv(t *testing.T) {
	tests := []struct {
		name string
		currentEnv []string
		env []string
		want []string
	}{
		{
			"Appends",
			[]string{"Aaa=a", "Bbb=b", "Ccc=c"},
			[]string{"Ddd=4"},
			[]string{ "Aaa=a", "Bbb=b", "Ccc=c", "Ddd=4"},
		},
		{
			"Appends and replaces conflict",
			[]string{"Aaa=a", "Bbb=b", "Ccc=c"},
			[]string{"Aaa=1", "Ccc=3", "Ddd=4"},
			[]string{"Bbb=b", "Aaa=1", "Ccc=3", "Ddd=4"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appendEnv(tt.currentEnv, tt.env...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("appendEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}