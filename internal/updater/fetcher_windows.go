// +build windows

package updater

import "github.com/ActiveState/cli/internal/unarchiver"

func blobUnarchiver(blob []byte) *unarchiver.ZipBlob {
	return unarchiver.NewZipBlob(blob)
}
