//go:build windows
// +build windows

package orgkey

import "os"

// checkCacheMode is a no-op on Windows, where POSIX permission bits do not
// apply; the cache file is written to the owner's config directory.
func checkCacheMode(info os.FileInfo) error {
	return nil
}
