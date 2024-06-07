package hash

import (
	"crypto/sha1"
	"fmt"
	"io"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/multilog"
)

// ShortHash will return the first 4 bytes in base16 of the sha1 sum of the provided data.
//
// For example:
//
//	  ShortHash("ActiveState-TestProject-python3")
//		 => e784c7e0
//
// This is useful for creating a shortened namespace for language installations.
func ShortHash(data ...string) string {
	h := sha1.New()
	_, err := io.WriteString(h, strings.Join(data, ""))
	if err != nil {
		// Error is very unlikely here, but we want to know if it does happen
		multilog.Error("Could not write to hash: %s", errs.JoinMessage(err))
	}
	return fmt.Sprintf("%x", h.Sum(nil)[:4])
}
