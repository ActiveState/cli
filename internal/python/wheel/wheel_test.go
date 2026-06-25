package wheel

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"
)

// makeTree writes files (keyed by slash-relative path) under root, creating
// parent directories as needed.
func makeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, body := range files {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

// readWheel returns the entries of a wheel keyed by archive path.
func readWheel(t *testing.T, path string) map[string][]byte {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open wheel: %v", err)
	}
	defer zr.Close()
	out := map[string][]byte{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		out[f.Name] = b
	}
	return out
}

func TestPackProducesValidWheel(t *testing.T) {
	src := t.TempDir()
	makeTree(t, src, map[string]string{
		"mypkg/__init__.py":       "print('hi')\n",
		"mypkg/util.py":           "x = 1\n",
		"mypkg/data.txt":          "data\n",
		"pyproject.toml":          "[project]\nname = \"ignored\"\n",
		"mypkg/__pycache__/u.pyc": "junk",
		"__pycache__/top.pyc":     "junk",
		".git/config":             "[core]\n",
	})

	out := t.TempDir()
	wheelPath, err := Pack(src, Metadata{Name: "My.Pkg-Name", Version: "1.0", Summary: "a pkg"}, out)
	if err != nil {
		t.Fatalf("Pack: %v", err)
	}

	if got, want := filepath.Base(wheelPath), "my_pkg_name-1.0-py3-none-any.whl"; got != want {
		t.Errorf("wheel filename = %q, want %q", got, want)
	}

	entries := readWheel(t, wheelPath)
	di := "my_pkg_name-1.0.dist-info"

	// Source files present at the wheel root; cruft excluded.
	for _, want := range []string{"mypkg/__init__.py", "mypkg/util.py", "mypkg/data.txt"} {
		if _, ok := entries[want]; !ok {
			t.Errorf("missing expected entry %q", want)
		}
	}
	for _, excluded := range []string{"pyproject.toml", "mypkg/__pycache__/u.pyc", "__pycache__/top.pyc", ".git/config"} {
		if _, ok := entries[excluded]; ok {
			t.Errorf("entry %q should have been excluded", excluded)
		}
	}

	// METADATA / WHEEL contents.
	meta := string(entries[di+"/METADATA"])
	for _, want := range []string{"Metadata-Version: 2.1", "Name: My.Pkg-Name", "Version: 1.0", "Summary: a pkg"} {
		if !bytes.Contains([]byte(meta), []byte(want)) {
			t.Errorf("METADATA missing %q; got:\n%s", want, meta)
		}
	}
	wheelFile := string(entries[di+"/WHEEL"])
	for _, want := range []string{"Root-Is-Purelib: true", "Tag: py3-none-any", "Wheel-Version: 1.0"} {
		if !bytes.Contains([]byte(wheelFile), []byte(want)) {
			t.Errorf("WHEEL missing %q; got:\n%s", want, wheelFile)
		}
	}

	// RECORD: every listed file's hash/size matches, RECORD itself unhashed.
	rec, ok := entries[di+"/RECORD"]
	if !ok {
		t.Fatal("missing RECORD")
	}
	rows, err := csv.NewReader(bytes.NewReader(rec)).ReadAll()
	if err != nil {
		t.Fatalf("parse RECORD: %v", err)
	}
	sawSelf := false
	for _, row := range rows {
		if len(row) != 3 {
			t.Fatalf("RECORD row has %d fields: %v", len(row), row)
		}
		name, hash, size := row[0], row[1], row[2]
		if name == di+"/RECORD" {
			sawSelf = true
			if hash != "" || size != "" {
				t.Errorf("RECORD self-row should have empty hash/size, got %q,%q", hash, size)
			}
			continue
		}
		content, ok := entries[name]
		if !ok {
			t.Errorf("RECORD lists %q which is not in the wheel", name)
			continue
		}
		sum := sha256.Sum256(content)
		wantHash := "sha256=" + base64.RawURLEncoding.EncodeToString(sum[:])
		if hash != wantHash {
			t.Errorf("%s: RECORD hash %q, want %q", name, hash, wantHash)
		}
		if size != strconv.Itoa(len(content)) {
			t.Errorf("%s: RECORD size %q, want %d", name, size, len(content))
		}
	}
	if !sawSelf {
		t.Error("RECORD does not list itself")
	}
}

func TestPackIsDeterministic(t *testing.T) {
	files := map[string]string{
		"mypkg/__init__.py": "print('hi')\n",
		"mypkg/util.py":     "x = 1\n",
	}
	meta := Metadata{Name: "pkg", Version: "2.0"}

	src1 := t.TempDir()
	makeTree(t, src1, files)
	out1 := t.TempDir()
	p1, err := Pack(src1, meta, out1)
	if err != nil {
		t.Fatal(err)
	}

	// A second tree with identical content but perturbed mtimes.
	src2 := t.TempDir()
	makeTree(t, src2, files)
	skew := time.Now().Add(-72 * time.Hour)
	_ = filepath.Walk(src2, func(p string, info os.FileInfo, err error) error {
		if err == nil {
			_ = os.Chtimes(p, skew, skew)
		}
		return nil
	})
	out2 := t.TempDir()
	p2, err := Pack(src2, meta, out2)
	if err != nil {
		t.Fatal(err)
	}

	b1, err := os.ReadFile(p1)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := os.ReadFile(p2)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b1, b2) {
		t.Errorf("wheels differ across runs (%d vs %d bytes)", len(b1), len(b2))
	}
}

func TestPackRejectsNativeContent(t *testing.T) {
	// Includes uppercase variants: extension matching must be case-insensitive
	// because Windows filesystems are.
	for _, ext := range []string{".so", ".pyd", ".dll", ".dylib", ".SO", ".DLL"} {
		src := t.TempDir()
		makeTree(t, src, map[string]string{
			"mypkg/__init__.py":    "x = 1\n",
			"mypkg/_speedup" + ext: "\x7fELF",
		})
		_, err := Pack(src, Metadata{Name: "pkg", Version: "1.0"}, t.TempDir())
		if !errors.Is(err, ErrNativeContent) {
			t.Errorf("%s: error = %v, want ErrNativeContent", ext, err)
		}
	}
}

func TestPackRejectsNoPython(t *testing.T) {
	t.Run("empty dir", func(t *testing.T) {
		_, err := Pack(t.TempDir(), Metadata{Name: "pkg", Version: "1.0"}, t.TempDir())
		if !errors.Is(err, ErrNoPythonFiles) {
			t.Errorf("error = %v, want ErrNoPythonFiles", err)
		}
	})
	t.Run("no python files", func(t *testing.T) {
		src := t.TempDir()
		makeTree(t, src, map[string]string{"docs/readme.txt": "hello\n"})
		_, err := Pack(src, Metadata{Name: "pkg", Version: "1.0"}, t.TempDir())
		if !errors.Is(err, ErrNoPythonFiles) {
			t.Errorf("error = %v, want ErrNoPythonFiles", err)
		}
	})
}

func TestPackNoPartialOutputOnFailure(t *testing.T) {
	src := t.TempDir()
	makeTree(t, src, map[string]string{
		"mypkg/__init__.py": "x = 1\n",
		"mypkg/ext.so":      "native",
	})
	out := t.TempDir()
	if _, err := Pack(src, Metadata{Name: "pkg", Version: "1.0"}, out); err == nil {
		t.Fatal("expected failure")
	}
	entries, _ := os.ReadDir(out)
	if len(entries) != 0 {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		sort.Strings(names)
		t.Errorf("output dir not clean after failure: %v", names)
	}
}
