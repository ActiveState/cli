package wheelinstall

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// makeWheel writes a zip (wheel) containing the given name->body entries and
// returns its path.
func makeWheel(t *testing.T, dir string, entries map[string]string) string {
	t.Helper()
	wheelPath := filepath.Join(dir, "greeting-1.0-py3-none-any.whl")
	f, err := os.Create(wheelPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for name, body := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return wheelPath
}

func TestInstall(t *testing.T) {
	t.Run("extracts the package and records the installer", func(t *testing.T) {
		dir := t.TempDir()
		wheel := makeWheel(t, dir, map[string]string{
			"greeting/__init__.py":            "print('hi')\n",
			"greeting-1.0.dist-info/METADATA": "Name: greeting\nVersion: 1.0\n",
			"greeting-1.0.dist-info/RECORD":   "",
		})

		site := filepath.Join(dir, "site-packages")
		if err := Install(wheel, site); err != nil {
			t.Fatalf("Install: %v", err)
		}

		if got, _ := os.ReadFile(filepath.Join(site, "greeting", "__init__.py")); string(got) != "print('hi')\n" {
			t.Errorf("package not extracted: got %q", got)
		}
		installer, err := os.ReadFile(filepath.Join(site, "greeting-1.0.dist-info", "INSTALLER"))
		if err != nil {
			t.Fatalf("INSTALLER not written: %v", err)
		}
		if string(installer) != installerName+"\n" {
			t.Errorf("INSTALLER = %q, want %q", installer, installerName+"\n")
		}
	})

	t.Run("a wheel without a .dist-info fails closed", func(t *testing.T) {
		dir := t.TempDir()
		wheel := makeWheel(t, dir, map[string]string{"greeting/__init__.py": "x\n"})
		if err := Install(wheel, filepath.Join(dir, "site-packages")); err == nil {
			t.Error("expected an error for a wheel without a .dist-info directory")
		}
	})
}
