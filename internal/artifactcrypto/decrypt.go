package artifactcrypto

import (
	"crypto/cipher"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
)

// Decrypt reads an encrypted payload from src, verifies it under the supplied
// 32-byte AES-256 key, and writes the recovered plaintext to destPath.
//
// Decrypt fails closed: it streams into a sibling temporary file and renames it
// onto destPath only after the entire payload verifies. On any failure the
// temporary file is removed and destPath is left untouched.
func Decrypt(src io.Reader, destPath string, key []byte) (rerr error) {
	header, err := ParseHeader(src)
	if err != nil {
		return errs.Wrap(err, "unable to parse header")
	}
	if err := header.CheckKey(key); err != nil {
		return errs.Wrap(err, "unable to verify key") // body never read
	}
	gcm, err := newGCM(key)
	if err != nil {
		return errs.Wrap(err, "unable to initialize decryption")
	}
	headerHash := header.headerHash()

	// A sibling temp file keeps the rename onto destPath atomic on the same filesystem.
	tmp, err := os.CreateTemp(filepath.Dir(destPath), filepath.Base(destPath)+".tmp-*")
	if err != nil {
		return errs.Wrap(err, "creating temp output")
	}
	tmpName := tmp.Name()
	defer func() {
		if rerr == nil {
			return
		}
		// Discard the partial output on failure, surfacing any cleanup errors.
		// tmp is nil once it has already been closed for the rename.
		if tmp != nil {
			if err := tmp.Close(); err != nil {
				rerr = errs.Pack(rerr, errs.Wrap(err, "closing temp output"))
			}
		}
		if err := os.Remove(tmpName); err != nil {
			rerr = errs.Pack(rerr, errs.Wrap(err, "removing temp output"))
		}
	}()

	if err := decryptBody(src, tmp, gcm, headerHash, header.ChunkSize); err != nil {
		return errs.Wrap(err, "unable to decrypt body")
	}

	// Close before renaming, and clear tmp so the deferred cleanup neither
	// double-closes nor runs at all on success.
	cerr := tmp.Close()
	tmp = nil
	if cerr != nil {
		return errs.Wrap(cerr, "closing temp output")
	}
	if err := os.Rename(tmpName, destPath); err != nil {
		return errs.Wrap(err, "finalizing output")
	}
	return nil
}

// decryptBody streams the chunk sequence from src, authenticating each chunk and
// writing its plaintext to dst. It returns an error (and writes no further
// plaintext) on the first authentication or framing failure, and requires that
// the stream end exactly after an authenticated final chunk.
func decryptBody(src io.Reader, dst io.Writer, gcm cipher.AEAD, headerHash [sha256.Size]byte, chunkSize uint32) error {
	maxCT := int(chunkSize) + tagSize
	var index uint64
	for {
		var frame [1 + nonceSize + 4]byte
		_, err := io.ReadFull(src, frame[:])
		if errors.Is(err, io.EOF) {
			// Clean end at a chunk boundary, but we never saw a final chunk.
			return ErrTruncated
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return ErrTruncated
		}
		if err != nil {
			return errs.Wrap(err, "reading chunk frame")
		}

		finalFlag := frame[0]
		if finalFlag != 0 && finalFlag != 1 {
			return ErrCorruptPayload
		}
		nonce := frame[1 : 1+nonceSize]
		ctLen := int(binary.BigEndian.Uint32(frame[1+nonceSize:]))
		if ctLen < tagSize || ctLen > maxCT {
			return ErrCorruptPayload
		}

		ct := make([]byte, ctLen)
		if _, err := io.ReadFull(src, ct); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return ErrTruncated
			}
			return errs.Wrap(err, "reading chunk body")
		}

		pt, err := gcm.Open(nil, nonce, ct, makeAAD(headerHash, index, finalFlag == 1))
		if err != nil {
			return ErrCorruptPayload // tamper, reorder, wrong final flag, or wrong key
		}
		if _, err := dst.Write(pt); err != nil {
			return errs.Wrap(err, "writing plaintext")
		}
		index++

		if finalFlag == 1 {
			// Reject any trailing data after the final chunk.
			var extra [1]byte
			if _, err := io.ReadFull(src, extra[:]); !errors.Is(err, io.EOF) {
				return ErrCorruptPayload
			}
			return nil
		}
	}
}
