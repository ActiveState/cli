// +build windows

package fileutils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hectane/go-acl"
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

func Test_IsWritable_File(t *testing.T) {
	file, err := WriteTempFile(
		"", t.Name(), []byte("Some data"), 0777,
	)
	if err != nil {
		t.Error(err)
	}

	if IsWritable(file) != true {
		t.Fatal("File should be writable")
	}

	err := acl.Chmod(file, 0444)
	if err != nil {
		t.Error(err)
	}

	if IsWritable(file) != false {
		t.Fatal("File should no longer be writable")
	}
}

func Test_IsWritable_Dir(t *testing.T) {
	dir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		t.Error(err)
	}

	if IsWritable(dir) != true {
		t.Fatal("Dir should be writable")
	}

	err = acl.Chmod(dir, 0444)
	if err != nil {
		t.Error(err)
	}

	if IsWritable(dir) != false {
		t.Fatal("Dir should no longer be writable")
	}
}

func Test_IsWritable_ReadOnly(t *testing.T) {
	dir, err := ioutil.TempDir("", t.Name())
	if err != nil {
		t.Error(err)
	}

	if IsWritable(dir) != true {
		t.Fatal("Dir should be writable")
	}

	err = os.Chmod(dir, 0400)
	if err != nil {
		t.Error(err)
	}

	if IsWritable(dir) != false {
		t.Fatal("Dir should no longer be writable")
	}
}
