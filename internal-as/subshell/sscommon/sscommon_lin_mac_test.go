//go:build !windows
// +build !windows

package sscommon

import (
	"fmt"
	"reflect"
	"testing"
)

func TestEscapeEnv(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]string
		want  map[string]string
	}{
		{
			"Escapes Env",
			map[string]string{
				"k1": fmt.Sprintf("v1\"%sv1", lineBreak),
				"k2": "v2",
			},
			map[string]string{
				"k1": fmt.Sprintf(`v1\"%sv1`, lineBreakChar),
				"k2": `v2`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EscapeEnv(tt.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EscapeEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}
