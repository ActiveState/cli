package artifactcrypto

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "regenerate golden test vectors")

const goldenFile = "testdata/golden_v1.bin"

// seqReader is a deterministic byte source (0,1,2,...,255,0,...) that pins
// nonces so the golden payload is byte-for-byte reproducible.
type seqReader struct{ n byte }

func (s *seqReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = s.n
		s.n++
	}
	return len(p), nil
}

// Fixed inputs that keep the golden payload stable.
var (
	goldenKey       = bytes.Repeat([]byte{0x24}, KeySize)
	goldenPlaintext = []byte("the quick brown fox jumps over the lazy dog, twice over now")
	goldenKeyID     = "golden-key-id"
	goldenChunkSize = 16
)

func generateGolden(t *testing.T) []byte {
	t.Helper()
	prevRand, prevChunk := randReader, encChunkSize
	randReader, encChunkSize = &seqReader{}, goldenChunkSize
	t.Cleanup(func() { randReader, encChunkSize = prevRand, prevChunk })

	var buf bytes.Buffer
	if err := Encrypt(bytes.NewReader(goldenPlaintext), &buf, goldenKey, goldenKeyID); err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	return buf.Bytes()
}

// TestGoldenVectorV1 compares the encoder output against the committed golden
// payload byte-for-byte. Run `go test -update` to regenerate after an
// intentional format change.
func TestGoldenVectorV1(t *testing.T) {
	got := generateGolden(t)

	if *update {
		if err := os.MkdirAll(filepath.Dir(goldenFile), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenFile, got, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %d bytes to %s", len(got), goldenFile)
		return
	}

	want, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("reading golden vector (run `go test -update` to create it): %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("payload format drifted from golden vector (%d bytes vs %d); run `go test -update` if intentional", len(got), len(want))
	}
}

// TestGoldenVectorDecrypts confirms the committed golden payload decrypts to the
// expected plaintext and parses to the expected header.
func TestGoldenVectorDecrypts(t *testing.T) {
	payload, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Skipf("golden vector not present (run `go test -update`): %v", err)
	}

	h, err := ParseHeader(bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("ParseHeader: %v", err)
	}
	if h.Version != formatVersion || h.KeyID != goldenKeyID || h.Fingerprint != Fingerprint(goldenKey) {
		t.Fatalf("unexpected header: %+v", h)
	}

	dest := filepath.Join(t.TempDir(), "golden.out")
	if err := Decrypt(bytes.NewReader(payload), dest, goldenKey); err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, goldenPlaintext) {
		t.Fatalf("golden decrypt mismatch: got %q", got)
	}
}
