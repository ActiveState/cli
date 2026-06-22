//go:build !windows
// +build !windows

package orgkey

import (
	"os"

	"github.com/ActiveState/cli/internal/errs"
)

// checkCacheMode rejects a cache file that is readable or writable by anyone
// other than the owner (anything beyond u+rw).
func checkCacheMode(info os.FileInfo) error {
	if info.Mode()&0177 != 0 {
		return errs.New("cache file %q must be mode 0600", info.Name())
	}
	return nil
}
