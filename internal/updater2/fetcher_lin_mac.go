// +build linux darwin

package updater2

import "github.com/ActiveState/cli/internal/unarchiver"

func blobUnarchiver(blob []byte) *unarchiver.TarGzBlob {
	return unarchiver.NewTarGzBlob(blob)
}
