package artifactcrypto

import (
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"io"

	"github.com/ActiveState/cli/internal/errs"
)

// Encrypt reads the artifact from src and writes the encrypted payload to dst,
// sealed under the supplied 32-byte AES-256 key. keyID is recorded in the
// header. Encryption streams with memory bounded by the chunk size. A zero-byte
// input produces a single empty final chunk.
func Encrypt(src io.Reader, dst io.Writer, key []byte, keyID string) error {
	gcm, err := newGCM(key)
	if err != nil {
		return errs.Wrap(err, "unable to initialize encryption")
	}

	raw := serializeHeader(keyID, Fingerprint(key), uint32(encChunkSize))
	if len(raw) > maxHeaderLen {
		return ErrHeaderTooLarge
	}
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(raw)))
	if _, err := dst.Write(lenBuf[:]); err != nil {
		return errs.Wrap(err, "writing header length")
	}
	if _, err := dst.Write(raw); err != nil {
		return errs.Wrap(err, "writing header")
	}
	headerHash := sha256.Sum256(raw)

	// Ping-pong buffers: we read one chunk ahead so we know whether the chunk
	// in hand is the final one (the last chunk may be short, including empty).
	bufA := make([]byte, encChunkSize)
	bufB := make([]byte, encChunkSize)

	cur, curN, err := readChunk(src, bufA)
	if err != nil {
		return errs.Wrap(err, "reading plaintext")
	}
	var index uint64
	for {
		next, nextN, err := readChunk(src, bufB)
		if err != nil {
			return errs.Wrap(err, "reading plaintext")
		}
		final := nextN == 0 // nothing more to read: the chunk in hand is final
		if err := sealChunk(dst, gcm, headerHash, index, final, cur[:curN]); err != nil {
			return errs.Wrap(err, "unable to seal chunk")
		}
		if final {
			return nil
		}
		cur, bufB = next, cur // reuse the buffer we just drained for the next read-ahead
		curN = nextN
		index++
	}
}

// readChunk fills buf with up to len(buf) bytes, treating EOF (with or without
// a partial fill) as a normal short read rather than an error. It returns the
// number of bytes read; n == 0 means the stream is exhausted.
func readChunk(r io.Reader, buf []byte) ([]byte, int, error) {
	n, err := io.ReadFull(r, buf)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return buf, n, nil
	}
	if err != nil {
		return buf, n, err
	}
	return buf, n, nil
}

// sealChunk seals one plaintext chunk under a fresh nonce and writes its frame:
// finalFlag(1) || nonce(12) || ciphertextLen(uint32) || ciphertext+tag.
func sealChunk(dst io.Writer, gcm cipher.AEAD, headerHash [sha256.Size]byte, index uint64, final bool, plaintext []byte) error {
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(randReader, nonce); err != nil {
		return errs.Wrap(err, "generating nonce")
	}
	ct := gcm.Seal(nil, nonce, plaintext, makeAAD(headerHash, index, final))

	var frame [1 + nonceSize + 4]byte
	frame[0] = finalByte(final)
	copy(frame[1:1+nonceSize], nonce)
	binary.BigEndian.PutUint32(frame[1+nonceSize:], uint32(len(ct)))
	if _, err := dst.Write(frame[:]); err != nil {
		return errs.Wrap(err, "writing chunk frame")
	}
	if _, err := dst.Write(ct); err != nil {
		return errs.Wrap(err, "writing chunk body")
	}
	return nil
}
