package wheel

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writePyproject(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestResolveMetadata(t *testing.T) {
	t.Run("pyproject fills empty fields", func(t *testing.T) {
		dir := t.TempDir()
		writePyproject(t, dir, "[project]\nname = \"proj\"\nversion = \"3.1\"\ndescription = \"from toml\"\n")
		res, err := resolveMetadata(dir, Metadata{})
		if err != nil {
			t.Fatal(err)
		}
		if res.Name != "proj" || res.Version != "3.1" || res.Summary != "from toml" {
			t.Errorf("got %+v", res)
		}
	})

	t.Run("caller overrides pyproject", func(t *testing.T) {
		dir := t.TempDir()
		writePyproject(t, dir, "[project]\nname = \"proj\"\nversion = \"3.1\"\n")
		res, err := resolveMetadata(dir, Metadata{Name: "override", Version: "9.9"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Name != "override" || res.Version != "9.9" {
			t.Errorf("override did not win: %+v", res)
		}
	})

	t.Run("missing name and version errors", func(t *testing.T) {
		dir := t.TempDir() // no pyproject.toml
		if _, err := resolveMetadata(dir, Metadata{Name: "only-name"}); !errors.Is(err, ErrMissingMetadata) {
			t.Errorf("error = %v, want ErrMissingMetadata", err)
		}
	})
}

func TestNormalizeName(t *testing.T) {
	cases := map[string]string{
		"My.Pkg-Name": "my_pkg_name",
		"Flask":       "flask",
		"a--b__c.d":   "a_b_c_d",
		"already_ok":  "already_ok",
	}
	for in, want := range cases {
		if got := normalizeName(in); got != want {
			t.Errorf("normalizeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEscapeVersion(t *testing.T) {
	cases := map[string]string{
		"1.0":       "1.0",
		"1.0+local": "1.0_local",
		"2.0-rc1":   "2.0_rc1",
	}
	for in, want := range cases {
		if got := escapeVersion(in); got != want {
			t.Errorf("escapeVersion(%q) = %q, want %q", in, got, want)
		}
	}
}
