//go:build windows

package deploy

import (
	"path/filepath"
	"testing"
)

func Test_symlinkName(t *testing.T) {
	name := symlinkTargetPath(filepath.FromSlash("/d/e/"), filepath.FromSlash("/a/test.exe.exe"))
	expected := filepath.FromSlash("/d/e/test.exe.lnk")
	if name != expected {
		t.Errorf("expected = %s, got %s", expected, name)
	}
}
