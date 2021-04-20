// +build linux darwin

package updater

import "github.com/ActiveState/cli/internal/unarchiver"

func blobUnarchiver(blob []byte) *unarchiver.TarGzBlob {
	return unarchiver.NewTarGzBlob(blob)
}
