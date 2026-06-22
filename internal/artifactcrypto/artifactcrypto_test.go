package artifactcrypto

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// testKey is a fixed 32-byte AES-256 key used across tests.
var testKey = bytes.Repeat([]byte{0x42}, KeySize)

// withChunkSize temporarily shrinks the encryption chunk size so multi-chunk
// behavior can be exercised on small inputs.
func withChunkSize(t *testing.T, size int) {
	t.Helper()
	prev := encChunkSize
	encChunkSize = size
	t.Cleanup(func() { encChunkSize = prev })
}

// encryptToBytes encrypts plaintext under testKey and returns the payload.
func encryptToBytes(t *testing.T, plaintext []byte, keyID string) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := Encrypt(bytes.NewReader(plaintext), &buf, testKey, keyID); err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	return buf.Bytes()
}

// decryptToBytes decrypts payload to a temp file under testKey and returns the
// recovered plaintext along with the destination path.
func decryptToBytes(t *testing.T, payload, key []byte) ([]byte, string, error) {
	t.Helper()
	dest := filepath.Join(t.TempDir(), "out.bin")
	err := Decrypt(bytes.NewReader(payload), dest, key)
	if err != nil {
		return nil, dest, err
	}
	got, readErr := os.ReadFile(dest)
	if readErr != nil {
		t.Fatalf("reading decrypted output: %v", readErr)
	}
	return got, dest, nil
}

func TestRoundTrip(t *testing.T) {
	withChunkSize(t, 64)
	sizes := []int{
		0,        // empty
		1,        // sub-chunk
		63,       // just under a chunk
		64,       // exactly one chunk
		65,       // one chunk + 1
		128,      // exactly two chunks
		64*3 + 7, // multiple chunks + partial
	}
	for _, n := range sizes {
		plaintext := make([]byte, n)
		if _, err := rand.Read(plaintext); err != nil {
			t.Fatal(err)
		}
		payload := encryptToBytes(t, plaintext, "test-key-01")
		got, _, err := decryptToBytes(t, payload, testKey)
		if err != nil {
			t.Fatalf("size %d: Decrypt: %v", n, err)
		}
		if !bytes.Equal(got, plaintext) {
			t.Fatalf("size %d: round-trip mismatch", n)
		}
	}
}

// TestRoundTripPreservesZip checks that a ZIP survives an Encrypt/Decrypt
// round-trip byte-for-byte and still opens as a valid archive.
func TestRoundTripPreservesZip(t *testing.T) {
	withChunkSize(t, 64) // exercise multi-chunk on a small archive

	var zbuf bytes.Buffer
	zw := zip.NewWriter(&zbuf)
	files := map[string]string{
		"pkg/__init__.py":            "print('hello from a private wheel')\n",
		"pkg/module.py":              bytes.NewBuffer(make([]byte, 300)).String(),
		"pkg-1.0.dist-info/METADATA": "Name: pkg\nVersion: 1.0\n",
	}
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, body); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	original := zbuf.Bytes()

	payload := encryptToBytes(t, original, "wheel-key")
	got, dest, err := decryptToBytes(t, payload, testKey)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Fatal("decrypted bytes differ from the original wheel")
	}
	zr, err := zip.OpenReader(dest)
	if err != nil {
		t.Fatalf("decrypted file is not a valid zip: %v", err)
	}
	defer zr.Close()
	if len(zr.File) != len(files) {
		t.Fatalf("zip entry count = %d, want %d", len(zr.File), len(files))
	}
}

func TestFingerprint(t *testing.T) {
	fp := Fingerprint(testKey)
	if len(fp) != len("sha256:")+64 {
		t.Fatalf("unexpected fingerprint length: %q", fp)
	}
	if fp[:7] != "sha256:" {
		t.Fatalf("fingerprint missing sha256 prefix: %q", fp)
	}
	// A different key yields a different fingerprint.
	other := bytes.Repeat([]byte{0x01}, KeySize)
	if Fingerprint(other) == fp {
		t.Fatal("distinct keys produced identical fingerprints")
	}
}

func TestWrongKeyRejectedBeforeBody(t *testing.T) {
	payload := encryptToBytes(t, []byte("secret wheel contents"), "kid")

	// The reader errors past the header, so reaching the body fails the test.
	wrongKey := bytes.Repeat([]byte{0x99}, KeySize)
	headerLen := binary.BigEndian.Uint32(payload[:4])
	headerOnly := payload[:4+int(headerLen)]
	src := io.MultiReader(bytes.NewReader(headerOnly), failingReader{})

	dest := filepath.Join(t.TempDir(), "out.bin")
	err := Decrypt(src, dest, wrongKey)
	if !errors.Is(err, ErrWrongKey) {
		t.Fatalf("expected ErrWrongKey, got %v", err)
	}
	assertNoOutput(t, dest)
}

func TestParseHeaderAndCheckKey(t *testing.T) {
	payload := encryptToBytes(t, []byte("hello"), "my-key-id")
	h, err := ParseHeader(bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("ParseHeader: %v", err)
	}
	if h.Version != formatVersion {
		t.Errorf("version = %d, want %d", h.Version, formatVersion)
	}
	if h.KeyID != "my-key-id" {
		t.Errorf("keyID = %q, want %q", h.KeyID, "my-key-id")
	}
	if h.Fingerprint != Fingerprint(testKey) {
		t.Errorf("fingerprint = %q, want %q", h.Fingerprint, Fingerprint(testKey))
	}
	if err := h.CheckKey(testKey); err != nil {
		t.Errorf("CheckKey(correct) = %v, want nil", err)
	}
	if err := h.CheckKey(bytes.Repeat([]byte{0x00}, KeySize)); !errors.Is(err, ErrWrongKey) {
		t.Errorf("CheckKey(wrong) = %v, want ErrWrongKey", err)
	}
}

func TestNonceUniquenessAcrossChunks(t *testing.T) {
	withChunkSize(t, 16)
	// Many chunks, so a reused nonce would show up as a duplicate below.
	const chunks = 200
	plaintext := make([]byte, 16*chunks)
	if _, err := rand.Read(plaintext); err != nil {
		t.Fatal(err)
	}
	payload := encryptToBytes(t, plaintext, "kid")

	nonces := extractNonces(t, payload)
	if len(nonces) < chunks {
		t.Fatalf("expected at least %d chunks, parsed %d", chunks, len(nonces))
	}
	seen := make(map[string]int, len(nonces))
	for i, n := range nonces {
		if prev, dup := seen[string(n)]; dup {
			t.Fatalf("nonce reused between chunk %d and chunk %d", prev, i)
		}
		seen[string(n)] = i
	}
}

func TestKeySizeValidation(t *testing.T) {
	short := make([]byte, 16)
	if err := Encrypt(bytes.NewReader(nil), io.Discard, short, "kid"); !errors.Is(err, ErrInvalidKeySize) {
		t.Errorf("Encrypt(short key) = %v, want ErrInvalidKeySize", err)
	}
	payload := encryptToBytes(t, []byte("x"), "kid")
	dest := filepath.Join(t.TempDir(), "out.bin")
	if err := Decrypt(bytes.NewReader(payload), dest, short); !errors.Is(err, ErrInvalidKeySize) {
		t.Errorf("Decrypt(short key) = %v, want ErrInvalidKeySize", err)
	}
	assertNoOutput(t, dest)
}

// extractNonces walks the chunk frames of a payload and returns each chunk's
// nonce. It mirrors the on-wire framing the encoder writes.
func extractNonces(t *testing.T, payload []byte) [][]byte {
	t.Helper()
	off := 4 + int(binary.BigEndian.Uint32(payload[:4])) // skip header
	var nonces [][]byte
	for off < len(payload) {
		// frame: finalFlag(1) | nonce(12) | ctLen(4) | ct(ctLen)
		nonce := payload[off+1 : off+1+nonceSize]
		ctLen := int(binary.BigEndian.Uint32(payload[off+1+nonceSize : off+1+nonceSize+4]))
		nonces = append(nonces, append([]byte(nil), nonce...))
		final := payload[off] == 1
		off += 1 + nonceSize + 4 + ctLen
		if final {
			break
		}
	}
	return nonces
}

func assertNoOutput(t *testing.T, dest string) {
	t.Helper()
	if _, err := os.Stat(dest); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("destination %q exists after a failed decrypt (stat err = %v)", dest, err)
	}
	// No stray temp siblings should be left behind either.
	matches, _ := filepath.Glob(dest + ".tmp-*")
	if len(matches) != 0 {
		t.Errorf("temp files left behind after failed decrypt: %v", matches)
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("body should not have been read")
}
