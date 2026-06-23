package runtime

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	goruntime "runtime"
	"testing"

	"github.com/ActiveState/cli/internal/artifactcrypto"
	"github.com/go-openapi/strfmt"
)

func testOrgKey() []byte {
	k := make([]byte, artifactcrypto.KeySize)
	for i := range k {
		k[i] = byte(i + 1)
	}
	return k
}

func encryptToBytes(t *testing.T, plaintext, key []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := artifactcrypto.Encrypt(bytes.NewReader(plaintext), &buf, key, "kid"); err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	return buf.Bytes()
}

// makeTarGz builds a gzip-compressed tar archive. symlinks maps a link path to
// its (relative) target, exercising the archive format's symlink support.
func makeTarGz(t *testing.T, files, symlinks map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     name,
			Mode:     0644,
			Size:     int64(len(body)),
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(tw, body); err != nil {
			t.Fatal(err)
		}
	}
	for name, target := range symlinks {
		if err := tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeSymlink,
			Name:     name,
			Mode:     0777,
			Linkname: target,
		}); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestFindEncryptedPayload(t *testing.T) {
	key := testOrgKey()

	t.Run("plaintext only", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "runtime.json"), []byte(`{"installDir":"."}`))
		writeFile(t, filepath.Join(dir, "data.txt"), []byte("ordinary file"))
		got, err := findEncryptedPayload(dir)
		if err != nil {
			t.Fatal(err)
		}
		if got != "" {
			t.Errorf("found a payload in a plaintext dir: %q", got)
		}
	})

	t.Run("finds the encrypted file", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "runtime.json"), []byte(`{"installDir":"."}`))
		payload := filepath.Join(dir, "anything.bin")
		writeFile(t, payload, encryptToBytes(t, []byte("secret"), key))
		got, err := findEncryptedPayload(dir)
		if err != nil {
			t.Fatal(err)
		}
		if got != payload {
			t.Errorf("got %q, want %q", got, payload)
		}
	})
}

func TestDecryptPayload(t *testing.T) {
	key := testOrgKey()
	payload := makeTarGz(t,
		map[string]string{
			"pkg/__init__.py":            "print('private')\n",
			"pkg-1.0.dist-info/METADATA": "Name: pkg\n",
		},
		map[string]string{
			"pkg/alias.py": "__init__.py", // relative symlink, resolves inside the artifact dir
		},
	)

	t.Run("happy path", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "runtime.json"), []byte(`{"installDir":"."}`))
		writeFile(t, filepath.Join(dir, "payload"), encryptToBytes(t, payload, key))

		s := &setup{opts: &Opts{OrgKey: key}}
		outcome, err := s.decryptPayload("pkg", dir)
		if err != nil {
			t.Fatalf("decryptPayload: %v", err)
		}
		if outcome != decryptDone {
			t.Fatalf("outcome = %v, want decryptDone", outcome)
		}
		// Ciphertext is removed.
		if _, err := os.Stat(filepath.Join(dir, "payload")); !os.IsNotExist(err) {
			t.Error("ciphertext was not removed")
		}
		// Archive contents extracted in place; the cleartext runtime.json survives.
		if got, _ := os.ReadFile(filepath.Join(dir, "pkg", "__init__.py")); string(got) != "print('private')\n" {
			t.Errorf("payload not extracted: got %q", got)
		}
		if !exists(filepath.Join(dir, "runtime.json")) {
			t.Error("runtime.json was lost")
		}
		// The symlink survived the tar.gz round-trip as a symlink.
		if goruntime.GOOS != "windows" {
			info, err := os.Lstat(filepath.Join(dir, "pkg", "alias.py"))
			if err != nil {
				t.Errorf("symlink not extracted: %v", err)
			} else if info.Mode()&os.ModeSymlink == 0 {
				t.Errorf("alias.py is not a symlink: mode %v", info.Mode())
			}
		}
		// 0700 owner-only boundary on the decrypted artifact dir.
		if goruntime.GOOS != "windows" {
			if info, _ := os.Stat(dir); info.Mode().Perm() != 0700 {
				t.Errorf("artifact dir mode = %v, want 0700", info.Mode().Perm())
			}
		}
	})

	t.Run("missing key skips", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "payload"), encryptToBytes(t, payload, key))

		s := &setup{opts: &Opts{}} // no OrgKey
		outcome, err := s.decryptPayload("pkg", dir)
		if err != nil {
			t.Fatalf("decryptPayload: %v", err)
		}
		if outcome != decryptSkipped {
			t.Fatalf("outcome = %v, want decryptSkipped", outcome)
		}
	})

	t.Run("wrong key fails closed", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "payload"), encryptToBytes(t, payload, key))

		wrong := make([]byte, artifactcrypto.KeySize) // all zeros
		s := &setup{opts: &Opts{OrgKey: wrong}}
		_, err := s.decryptPayload("pkg", dir)
		if err == nil {
			t.Fatal("expected a wrong-key error, got nil")
		}
	})

	t.Run("plaintext artifact is untouched", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "runtime.json"), []byte(`{"installDir":"."}`))
		s := &setup{opts: &Opts{OrgKey: key}}
		outcome, err := s.decryptPayload("pkg", dir)
		if err != nil {
			t.Fatal(err)
		}
		if outcome != decryptNotEncrypted {
			t.Fatalf("outcome = %v, want decryptNotEncrypted", outcome)
		}
	})
}

func TestPrivateArtifactSurvivesEviction(t *testing.T) {
	d := &depot{
		config: depotConfig{
			Deployments: map[strfmt.UUID][]deployment{},
			Cache: map[strfmt.UUID]*artifactInfo{
				strfmt.UUID("private"):    {Size: 100 * MB, Private: true, LastAccessTime: 1},
				strfmt.UUID("old-public"): {Size: 100 * MB, LastAccessTime: 1},
			},
		},
		depotPath: t.TempDir(),
		artifacts: map[strfmt.UUID]struct{}{},
		cacheSize: 50 * MB, // under pressure: the non-private entry must be evicted
	}

	if err := d.removeStaleArtifacts(); err != nil {
		t.Fatalf("removeStaleArtifacts: %v", err)
	}
	if _, ok := d.config.Cache[strfmt.UUID("private")]; !ok {
		t.Error("private artifact was evicted under cache pressure")
	}
	if _, ok := d.config.Cache[strfmt.UUID("old-public")]; ok {
		t.Error("non-private artifact was not evicted")
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
