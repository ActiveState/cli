// Package artifactcrypto encrypts and decrypts artifacts under a caller-supplied
// 32-byte AES-256 key.
//
// An artifact is encrypted as a sequence of AES-256-GCM chunks. Each chunk is
// sealed with a fresh random 96-bit nonce, and the chunk index and a
// final-chunk flag are bound into the AEAD additional data so chunk reordering
// and truncation are detected. A header (magic marker, format version, key id,
// and key fingerprint) precedes the chunks; its bytes are hashed into every
// chunk's AAD, so altering or stripping the header makes decryption fail.
//
// The header records a key id and a SHA-256 fingerprint of the key, never the
// key bytes.
package artifactcrypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"

	"github.com/ActiveState/cli/internal/errs"
)

var (
	// ErrBadMagic indicates the payload does not begin with the v1 magic marker.
	ErrBadMagic = errs.New("not an encrypted artifact (bad magic marker)")
	// ErrUnsupportedVersion indicates a payload format version this build cannot read.
	ErrUnsupportedVersion = errs.New("unsupported payload format version")
	// ErrCorruptPayload indicates a tampered, reordered, or otherwise unauthentic payload.
	ErrCorruptPayload = errs.New("corrupt or tampered payload")
	// ErrTruncated indicates the payload ended before a final chunk was authenticated.
	ErrTruncated = errs.New("truncated payload (no authenticated final chunk)")
	// ErrWrongKey indicates the supplied key does not match the payload's key fingerprint.
	ErrWrongKey = errs.New("key does not match payload fingerprint")
	// ErrInvalidKeySize indicates the supplied key is not a 32-byte AES-256 key.
	ErrInvalidKeySize = errs.New("key must be 32 bytes (AES-256)")
)

const (
	// KeySize is the required key length in bytes (AES-256).
	KeySize = 32
	// nonceSize is the AES-GCM standard 96-bit nonce.
	nonceSize = 12
	// tagSize is the AES-GCM authentication tag length.
	tagSize = 16
	// formatVersion is the payload format version this build writes.
	formatVersion = 1
	// DefaultChunkSize is the plaintext size of every chunk except the last.
	DefaultChunkSize = 1 << 20 // 1 MiB
	// maxChunkSize is the largest chunk size accepted from a parsed header.
	maxChunkSize = 64 << 20 // 64 MiB
	// maxHeaderLen is the largest serialized header accepted from a stream.
	maxHeaderLen = 64 << 10 // 64 KiB
)

// magicMarker marks the start of an encrypted payload.
const magicMarker = "ActiveStateEncrypted"

// randReader is the source of nonce randomness; tests override it to produce
// deterministic output.
var randReader io.Reader = rand.Reader

// encChunkSize is the plaintext chunk size used when encrypting; tests shrink it
// to exercise multi-chunk paths on small inputs. The size used is recorded in
// the header, so decryption adapts to it.
var encChunkSize = DefaultChunkSize

// Header is a parsed payload header. It carries only public metadata, never key
// bytes.
type Header struct {
	Version     uint8
	ChunkSize   uint32
	KeyID       string
	Fingerprint string // "sha256:<hex>" over the raw key bytes

	raw []byte // exact serialized header bytes, bound into every chunk's AAD
}

// Fingerprint returns an identifier for a key as "sha256:<hex>".
func Fingerprint(key []byte) string {
	sum := sha256.Sum256(key)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// CheckKey reports whether key matches the fingerprint recorded in the header,
// returning ErrWrongKey if it does not.
func (h Header) CheckKey(key []byte) error {
	if len(key) != KeySize {
		return ErrInvalidKeySize
	}
	if Fingerprint(key) != h.Fingerprint {
		return ErrWrongKey
	}
	return nil
}

// IsEncrypted reports whether src begins with the v1 payload marker. It reads
// only the leading length prefix and magic, so a non-payload stream (or one too
// short to be a payload) returns false without error. A stream whose marker
// matches but whose body is malformed still returns true here; that is caught
// when the payload is actually parsed or decrypted, so detection and validation
// stay distinct.
func IsEncrypted(src io.Reader) (bool, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(src, lenBuf[:]); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return false, nil
		}
		return false, errs.Wrap(err, "reading payload prefix")
	}
	magic := make([]byte, len(magicMarker))
	if _, err := io.ReadFull(src, magic); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return false, nil
		}
		return false, errs.Wrap(err, "reading payload marker")
	}
	return string(magic) == magicMarker, nil
}

// ParseHeader reads the header from src, consuming exactly the header bytes and
// leaving src positioned at the first chunk.
func ParseHeader(src io.Reader) (Header, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(src, lenBuf[:]); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return Header{}, ErrBadMagic
		}
		return Header{}, errs.Wrap(err, "reading header length")
	}
	hdrLen := binary.BigEndian.Uint32(lenBuf[:])
	if hdrLen == 0 || hdrLen > maxHeaderLen {
		return Header{}, ErrCorruptPayload
	}
	raw := make([]byte, hdrLen)
	if _, err := io.ReadFull(src, raw); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return Header{}, ErrTruncated
		}
		return Header{}, errs.Wrap(err, "reading header")
	}
	return parseHeaderBytes(raw)
}

// parseHeaderBytes decodes the serialized header fields from raw. It is strict:
// any trailing bytes, a bad magic marker, an out-of-range chunk size, or an
// unreadable field is rejected.
func parseHeaderBytes(raw []byte) (Header, error) {
	r := bytes.NewReader(raw)

	magic := make([]byte, len(magicMarker))
	if _, err := io.ReadFull(r, magic); err != nil {
		return Header{}, ErrBadMagic
	}
	if string(magic) != magicMarker {
		return Header{}, ErrBadMagic
	}

	version, err := r.ReadByte()
	if err != nil {
		return Header{}, ErrCorruptPayload
	}
	if version != formatVersion {
		return Header{}, ErrUnsupportedVersion
	}

	var u32 [4]byte
	if _, err := io.ReadFull(r, u32[:]); err != nil {
		return Header{}, ErrCorruptPayload
	}
	chunkSize := binary.BigEndian.Uint32(u32[:])
	if chunkSize == 0 || chunkSize > maxChunkSize {
		return Header{}, ErrCorruptPayload
	}

	keyID, err := readLenPrefixed(r)
	if err != nil {
		return Header{}, ErrCorruptPayload
	}
	fingerprint, err := readLenPrefixed(r)
	if err != nil {
		return Header{}, ErrCorruptPayload
	}

	if r.Len() != 0 { // trailing bytes in the header are not allowed
		return Header{}, ErrCorruptPayload
	}

	return Header{
		Version:     version,
		ChunkSize:   chunkSize,
		KeyID:       string(keyID),
		Fingerprint: string(fingerprint),
		raw:         raw,
	}, nil
}

// serializeHeader returns the header byte sequence that is length-prefixed onto
// the wire and hashed into every chunk's AAD.
func serializeHeader(keyID, fingerprint string, chunkSize uint32) []byte {
	var b bytes.Buffer
	b.WriteString(magicMarker)
	b.WriteByte(formatVersion)
	var u32 [4]byte
	binary.BigEndian.PutUint32(u32[:], chunkSize)
	b.Write(u32[:])
	writeLenPrefixed(&b, []byte(keyID))
	writeLenPrefixed(&b, []byte(fingerprint))
	return b.Bytes()
}

// headerHash returns the SHA-256 of the serialized header, which is folded into
// every chunk's AAD.
func (h Header) headerHash() [sha256.Size]byte {
	return sha256.Sum256(h.raw)
}

// makeAAD builds the additional authenticated data for one chunk:
// headerHash || uint64(index) || finalFlag.
func makeAAD(headerHash [sha256.Size]byte, index uint64, final bool) []byte {
	var indexBytes [8]byte
	binary.BigEndian.PutUint64(indexBytes[:], index)

	aad := make([]byte, 0, len(headerHash)+len(indexBytes)+1)
	aad = append(aad, headerHash[:]...)
	aad = append(aad, indexBytes[:]...)
	aad = append(aad, finalByte(final))
	return aad
}

func finalByte(final bool) byte {
	if final {
		return 1
	}
	return 0
}

// newGCM validates the key length and returns an AES-256-GCM AEAD.
func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKeySize
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		// Unreachable: the key length is validated above.
		return nil, ErrInvalidKeySize
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errs.Wrap(err, "initializing GCM")
	}
	return gcm, nil
}

func writeLenPrefixed(b *bytes.Buffer, p []byte) {
	var l [2]byte
	binary.BigEndian.PutUint16(l[:], uint16(len(p)))
	b.Write(l[:])
	b.Write(p)
}

func readLenPrefixed(r *bytes.Reader) ([]byte, error) {
	var l [2]byte
	if _, err := io.ReadFull(r, l[:]); err != nil {
		return nil, err
	}
	n := int(binary.BigEndian.Uint16(l[:]))
	p := make([]byte, n)
	if _, err := io.ReadFull(r, p); err != nil {
		return nil, err
	}
	return p, nil
}
