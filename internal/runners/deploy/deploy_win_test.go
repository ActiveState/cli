// +build windows

package deploy

import (
	"path/filepath"
	"testing"
)

func Test_linkTarget(t *testing.T) {
	target := linkTarget(filepath.FromSlash("/d/e/"), filepath.FromSlash("/a/test.exe.exe"))
	expected := filepath.FromSlash("/d/e/test.exe.lnk")
	if target != expected {
		t.Errorf("expected = %s, got %s", expected, target)
	}
}

func Test_shouldOverwriteSymlink(t *testing.T) {

	symlinkedFiles := map[string]string{
		fileNameBase("test-a.exe"): filepath.FromSlash("/a/test-a.bat"),
	}

	pathExt := []string{".COM", ".BaT", ".exE"}

	tests := []struct {
		name      string
		path      string
		overwrite bool
		want      bool
		allowed   bool
	}{
		{"new executable (forced)", filepath.FromSlash("/a/new.exe"), true, true, true},
		{"new executable (disallowed)", filepath.FromSlash("/a/new.exe"), false, true, false},
		{"higher priority (forced)", filepath.FromSlash("/a/test-a.com"), true, true, true},
		{"higher priority (disallowed)", filepath.FromSlash("/a/test-a.com"), false, true, true},
		{"lower priority (forced)", filepath.FromSlash("/a/test-a.bat"), true, false, false},
		{"lower priority (disallowed)", filepath.FromSlash("/a/test-a.bat"), false, false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(tt *testing.T) {
			doOverwrite, allowed := shouldOverwrite(tc.overwrite, tc.path, symlinkedFiles, pathExt)
			if doOverwrite != tc.want {
				conditional := ""
				if !tc.want {
					conditional = "not"
				}
				t.Errorf("Expected that %s should %s overwrite existing symlink", tc.path, conditional)
			}
			if allowed != tc.allowed {
				conditional := ""
				if !tc.want {
					conditional = "not"
				}
				t.Errorf("Expected that %s should %s be allowed to overwrite existing symlink", tc.path, conditional)
			}
		})
	}
}
