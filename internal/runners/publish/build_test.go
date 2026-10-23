package publish

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/artifactcrypto"
	"github.com/ActiveState/cli/internal/python/wheel"
	"github.com/ActiveState/cli/pkg/platform/api/graphql/request"
)

func testKey() []byte {
	return bytes.Repeat([]byte{0x42}, artifactcrypto.KeySize)
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

// readTarGz returns the regular-file entries of a gzip'd tar keyed by name.
func readTarGz(t *testing.T, path string) map[string][]byte {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	out := map[string][]byte{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		b, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		out[filepath.ToSlash(hdr.Name)] = b
	}
	return out
}

func keysOf(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func TestRequireOrgNamespace(t *testing.T) {
	const owner = "myorg" // org namespace is private/myorg
	ok := []string{"private/myorg", "private/myorg/sub", "private/myorg/a/b"}
	bad := []string{"private/other", "private/myorg2", "public/myorg", "private", ""}
	for _, ns := range ok {
		if err := requireOrgNamespace(ns, owner); err != nil {
			t.Errorf("requireOrgNamespace(%q) = %v, want nil", ns, err)
		}
	}
	for _, ns := range bad {
		if err := requireOrgNamespace(ns, owner); err == nil {
			t.Errorf("requireOrgNamespace(%q) = nil, want an error", ns)
		}
	}
}

func TestBuildWrappedArtifact(t *testing.T) {
	src := t.TempDir()
	writeFile(t, filepath.Join(src, "mypkg", "__init__.py"), "print('hi')\n")
	writeFile(t, filepath.Join(src, "pyproject.toml"), "[project]\nname = \"My.Pkg\"\nversion = \"1.2.3\"\n")

	meta, err := wheel.ResolveMetadata(src, wheel.Metadata{})
	if err != nil {
		t.Fatal(err)
	}
	if meta.Name != "My.Pkg" || meta.Version != "1.2.3" {
		t.Errorf("resolved metadata = %+v, want name My.Pkg version 1.2.3", meta)
	}
	key := testKey()

	archivePath, cleanup, err := buildWrappedArtifact(src, *meta, key, "kid")
	if err != nil {
		t.Fatalf("buildWrappedArtifact: %v", err)
	}
	defer cleanup()

	// The wrapped archive contains exactly the ciphertext.
	entries := readTarGz(t, archivePath)
	if got, want := keysOf(entries), []string{"payload.enc"}; !equalStrings(got, want) {
		t.Fatalf("wrapped entries = %v, want %v", got, want)
	}

	// The payload is encrypted before it is wrapped.
	enc, err := artifactcrypto.IsEncrypted(bytes.NewReader(entries["payload.enc"]))
	if err != nil {
		t.Fatal(err)
	}
	if !enc {
		t.Error("payload.enc is not an encrypted payload")
	}

	// No plaintext survives in the build temp dir.
	for _, e := range mustReadDir(t, filepath.Dir(archivePath)) {
		if strings.HasSuffix(e.Name(), ".whl") || e.Name() == "payload.tar.gz" {
			t.Errorf("plaintext leftover in temp dir: %s", e.Name())
		}
	}

	// The ciphertext decrypts to a tar.gz holding the wheel.
	innerPath := filepath.Join(t.TempDir(), "inner.tar.gz")
	if err := artifactcrypto.Decrypt(bytes.NewReader(entries["payload.enc"]), innerPath, key); err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	foundWheel := false
	for _, name := range keysOf(readTarGz(t, innerPath)) {
		if strings.HasSuffix(name, ".whl") {
			foundWheel = true
		}
	}
	if !foundWheel {
		t.Error("decrypted payload does not contain the wheel")
	}

	// cleanup removes the build temp dir.
	cleanup()
	if _, err := os.Stat(filepath.Dir(archivePath)); !os.IsNotExist(err) {
		t.Error("cleanup did not remove the build temp dir")
	}
}

func mustReadDir(t *testing.T, dir string) []os.DirEntry {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestAddPrivateBuilderDependency(t *testing.T) {
	userDep := request.PublishVariableDep{Dependency: request.Dependency{Name: "requests", Namespace: "language/python"}}
	reqVars := request.PublishVariables{Dependencies: []request.PublishVariableDep{userDep}}

	addPrivateBuilderDependency(&reqVars)
	if len(reqVars.Dependencies) != 2 {
		t.Fatalf("expected user dependency plus private-builder, got %+v", reqVars.Dependencies)
	}
	if reqVars.Dependencies[0].Dependency != userDep.Dependency {
		t.Errorf("user dependency was not preserved: %+v", reqVars.Dependencies[0])
	}

	// An --edit revision inherits the private-builder; it must not be added twice.
	addPrivateBuilderDependency(&reqVars)
	if len(reqVars.Dependencies) != 2 {
		t.Errorf("private-builder was added twice: %+v", reqVars.Dependencies)
	}
}
