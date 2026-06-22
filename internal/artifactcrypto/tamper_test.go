package artifactcrypto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"path/filepath"
	"testing"
)

// TestTamperFailsClosed runs the full tamper suite: every mutation of a valid
// payload must cause Decrypt to fail and leave no plaintext at the destination.
func TestTamperFailsClosed(t *testing.T) {
	withChunkSize(t, 32)
	plaintext := bytes.Repeat([]byte("private wheel bytes "), 8) // ~160 bytes => several chunks
	keyID := "test-key-01"

	headerLen := func(p []byte) int { return int(binary.BigEndian.Uint32(p[:4])) }

	cases := []struct {
		name    string
		mutate  func(p []byte) []byte
		wantErr error // nil means "any error is acceptable, just must fail closed"
	}{
		{
			name: "header bit-flip (keyID region)",
			mutate: func(p []byte) []byte {
				// Flip a byte inside the keyID region of the header.
				idx := bytes.Index(p, []byte(keyID))
				p[idx] ^= 0xFF
				return p
			},
			wantErr: ErrCorruptPayload,
		},
		{
			name: "magic strip",
			mutate: func(p []byte) []byte {
				p[4] ^= 0xFF // first magic byte
				return p
			},
			wantErr: ErrBadMagic,
		},
		{
			name: "fingerprint strip",
			mutate: func(p []byte) []byte {
				// Last byte of the header is the last fingerprint char.
				p[4+headerLen(p)-1] ^= 0xFF
				return p
			},
			wantErr: ErrWrongKey, // CheckKey rejects before the body is read
		},
		{
			name: "body bit-flip",
			mutate: func(p []byte) []byte {
				// Flip a ciphertext byte deep in the first chunk.
				p[len(p)-5] ^= 0xFF
				return p
			},
			wantErr: ErrCorruptPayload,
		},
		{
			name: "chunk reorder",
			mutate: func(p []byte) []byte {
				return swapFirstTwoChunks(t, p)
			},
			wantErr: ErrCorruptPayload,
		},
		{
			name: "truncation (drop final chunk)",
			mutate: func(p []byte) []byte {
				return dropLastChunk(t, p)
			},
			wantErr: ErrTruncated,
		},
		{
			name: "truncation (mid-chunk-body cut)",
			mutate: func(p []byte) []byte {
				return p[:len(p)-3]
			},
			wantErr: ErrTruncated,
		},
		{
			name: "final flag cleared on last chunk",
			mutate: func(p []byte) []byte {
				off := lastChunkOffset(t, p)
				p[off] = 0 // was 1
				return p
			},
			wantErr: ErrCorruptPayload,
		},
		{
			name: "trailing data appended after final chunk",
			mutate: func(p []byte) []byte {
				return append(p, 0x00, 0x01, 0x02)
			},
			wantErr: ErrCorruptPayload,
		},
		{
			name: "ctLen inflated past chunk bound",
			mutate: func(p []byte) []byte {
				off := 4 + headerLen(p) // first chunk frame
				binary.BigEndian.PutUint32(p[off+1+nonceSize:], 0xFFFFFFFF)
				return p
			},
			wantErr: ErrCorruptPayload,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := encryptToBytes(t, plaintext, keyID)
			tampered := tc.mutate(append([]byte(nil), payload...))

			dest := filepath.Join(t.TempDir(), "out.bin")
			err := Decrypt(bytes.NewReader(tampered), dest, testKey)
			if err == nil {
				t.Fatal("tampered payload decrypted without error")
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			assertNoOutput(t, dest)
		})
	}
}

// chunkOffsets returns the byte offset of each chunk frame in a payload.
func chunkOffsets(t *testing.T, p []byte) []int {
	t.Helper()
	off := 4 + int(binary.BigEndian.Uint32(p[:4]))
	var offs []int
	for off < len(p) {
		offs = append(offs, off)
		ctLen := int(binary.BigEndian.Uint32(p[off+1+nonceSize : off+1+nonceSize+4]))
		final := p[off] == 1
		off += 1 + nonceSize + 4 + ctLen
		if final {
			break
		}
	}
	return offs
}

func lastChunkOffset(t *testing.T, p []byte) int {
	offs := chunkOffsets(t, p)
	if len(offs) == 0 {
		t.Fatal("no chunks found")
	}
	return offs[len(offs)-1]
}

func swapFirstTwoChunks(t *testing.T, p []byte) []byte {
	offs := chunkOffsets(t, p)
	if len(offs) < 2 {
		t.Fatalf("need at least 2 chunks to reorder, found %d", len(offs))
	}
	frame := func(off int) []byte {
		ctLen := int(binary.BigEndian.Uint32(p[off+1+nonceSize : off+1+nonceSize+4]))
		end := off + 1 + nonceSize + 4 + ctLen
		return p[off:end]
	}
	c0 := append([]byte(nil), frame(offs[0])...)
	c1 := append([]byte(nil), frame(offs[1])...)

	var out bytes.Buffer
	out.Write(p[:offs[0]]) // header + outer len
	out.Write(c1)
	out.Write(c0)
	out.Write(p[offs[1]+len(c1):]) // remaining chunks, if any
	return out.Bytes()
}

func dropLastChunk(t *testing.T, p []byte) []byte {
	off := lastChunkOffset(t, p)
	return p[:off]
}
