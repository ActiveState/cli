package orgkey

import (
	"io"

	"github.com/ActiveState/cli/internal/artifactcrypto"
	"github.com/ActiveState/cli/internal/errs"
)

// PreflightKey confirms that key matches the artifact whose encrypted payload
// begins at r, checking the payload header's fingerprint before any body is
// read, so publish can fail before upload and pull can fail before a body
// transfer. A mismatch returns an error matching artifactcrypto.ErrWrongKey.
func PreflightKey(r io.Reader, key []byte) error {
	header, err := artifactcrypto.ParseHeader(r)
	if err != nil {
		return errs.Wrap(err, "unable to read artifact header")
	}
	if err := header.CheckKey(key); err != nil {
		return errs.Wrap(err, "key does not match artifact")
	}
	return nil
}
