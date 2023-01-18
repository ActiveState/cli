package hash

import (
	"crypto/sha1"
	"fmt"
	"io"
	"strings"
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
	io.WriteString(h, strings.Join(data, ""))
	return fmt.Sprintf("%x", h.Sum(nil)[:4])
}
