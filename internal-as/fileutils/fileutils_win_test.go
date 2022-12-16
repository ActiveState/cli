// +build windows

package fileutils

import (
	"path/filepath"
	"testing"
)

func TestIsExecutable(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"exe lower case", filepath.FromSlash("/a/test-a.exe")},
		{"exe uppper case", filepath.FromSlash("/a/test-a.EXE")},
		{"bat lower case", filepath.FromSlash("/a/test-a.bat")},
		{"bat upper case", filepath.FromSlash("/a/test-a.BAT")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(tt *testing.T) {
			if !IsExecutable(tc.path) {
				tt.Errorf("expected %s to be executable", tc.path)
			}
		})
	}

	invalid := filepath.FromSlash("/d/e/test-a.txt")
	if IsExecutable(invalid) {
		t.Errorf("%s should not be executable", invalid)
	}
}

