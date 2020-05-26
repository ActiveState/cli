// +build windows

package deploy

import (
	"path/filepath"
	"testing"
)

func Test_symlinkName(t *testing.T) {
	name := symlinkName(filepath.FromSlash("/d/e/"), filepath.FromSlash("/a/test.exe.exe"))
	expected := filepath.FromSlash("/d/e/test.exe.lnk")
	if name != expected {
		t.Errorf("expected = %s, got %s", expected, name)
	}
}

func Test_shouldOverwriteSymlink(t *testing.T) {
	oldPath := "/a/test-a.bat"

	pathExt := []string{".COM", ".BaT", ".exE"}

	tests := []struct {
		name            string
		path            string
		shouldOverwrite bool
	}{
		{"higher priority", filepath.FromSlash("/a/test-a.com"), true},
		{"lower priority", filepath.FromSlash("/a/test-a.bat"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(tt *testing.T) {
			if shouldOverwriteSymlink(tc.path, oldPath, pathExt) != tc.shouldOverwrite {
				conditional := ""
				if !tc.shouldOverwrite {
					conditional = "not"
				}
				t.Errorf("Expected that %s should %s overwrite existing symlink", tc.path, conditional)
			}
		})
	}
}
