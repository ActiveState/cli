package unarchiver

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

type tarEntry struct {
	name     string
	typeflag byte
	linkname string
	body     string
}

// makeTarGz builds an in-memory tar.gz from the given entries, including
// deliberately malicious ones (the tar writer does not sanitize names).
func makeTarGz(t *testing.T, entries []tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		hdr := &tar.Header{Name: e.name, Typeflag: e.typeflag, Linkname: e.linkname, Mode: 0644}
		if e.typeflag == tar.TypeReg {
			hdr.Size = int64(len(e.body))
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write header %q: %v", e.name, err)
		}
		if e.typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatalf("write body %q: %v", e.name, err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestUntrustedSourceRejectsEscapes(t *testing.T) {
	tests := []struct {
		name    string
		entries []tarEntry
	}{
		{"path traversal", []tarEntry{{name: "../escape.txt", typeflag: tar.TypeReg, body: "x"}}},
		{"absolute symlink", []tarEntry{{name: "link", typeflag: tar.TypeSymlink, linkname: "/etc/passwd"}}},
		{"backslash-rooted symlink", []tarEntry{{name: "link", typeflag: tar.TypeSymlink, linkname: `\Windows\System32`}}},
		{"symlink escapes root", []tarEntry{{name: "sub/link", typeflag: tar.TypeSymlink, linkname: "../../outside"}}},
		{"hardlink escapes root", []tarEntry{{name: "link", typeflag: tar.TypeLink, linkname: "../outside"}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			dest := filepath.Join(root, "dest")
			ua := NewTarGz(WithUntrustedSource())
			err := ua.Unarchive(bytes.NewReader(makeTarGz(t, tc.entries)), dest)
			if err == nil {
				t.Fatal("expected rejection, got nil")
			}
			// Nothing should have been written outside dest.
			if entries, _ := os.ReadDir(root); len(entries) > 1 {
				t.Errorf("unexpected files written outside dest: %v", entries)
			}
		})
	}
}

func TestTrustedSourceAllowsEscape(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "dest")

	// Without WithUntrustedSource, an escaping entry extracts as before (the
	// entry lands at root/escape.txt, still inside the test's sandbox).
	ua := NewTarGz()
	err := ua.Unarchive(bytes.NewReader(makeTarGz(t, []tarEntry{
		{name: "../escape.txt", typeflag: tar.TypeReg, body: "trusted"},
	})), dest)
	if err != nil {
		t.Fatalf("default (trusted) extraction should not error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "escape.txt")); err != nil {
		t.Errorf("escaping entry should have been extracted: %v", err)
	}
}

func TestUntrustedSourceHappyPath(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "dest")
	ua := NewTarGz(WithUntrustedSource())
	err := ua.Unarchive(bytes.NewReader(makeTarGz(t, []tarEntry{
		{name: "dir/", typeflag: tar.TypeDir},
		{name: "dir/file.txt", typeflag: tar.TypeReg, body: "hello"},
	})), dest)
	if err != nil {
		t.Fatalf("happy path failed: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "dir", "file.txt"))
	if err != nil || string(got) != "hello" {
		t.Fatalf("file not extracted correctly: got %q err %v", got, err)
	}
}

func TestUntrustedSourceAllowsContainedLinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}
	dest := filepath.Join(t.TempDir(), "dest")
	ua := NewTarGz(WithUntrustedSource())
	err := ua.Unarchive(bytes.NewReader(makeTarGz(t, []tarEntry{
		{name: "file.txt", typeflag: tar.TypeReg, body: "data"},
		{name: "sym", typeflag: tar.TypeSymlink, linkname: "file.txt"}, // contained sibling
		{name: "hard", typeflag: tar.TypeLink, linkname: "file.txt"},   // contained target
	})), dest)
	if err != nil {
		t.Fatalf("contained links should extract: %v", err)
	}
	for _, name := range []string{"file.txt", "sym", "hard"} {
		if _, err := os.Lstat(filepath.Join(dest, name)); err != nil {
			t.Errorf("expected %q to be extracted: %v", name, err)
		}
	}
}

func TestIsContainedPath(t *testing.T) {
	root := filepath.Clean(t.TempDir())
	tests := []struct {
		name      string
		contained bool
	}{
		{"file.txt", true},
		{"sub/file.txt", true},
		{"../escape.txt", false},
		{"sub/../../escape.txt", false},
	}
	for _, tc := range tests {
		path := filepath.Join(root, tc.name)
		if got := isContainedPath(root, path); got != tc.contained {
			t.Errorf("isContainedPath(root, join(%q)) = %v, want %v", tc.name, got, tc.contained)
		}
	}
}
