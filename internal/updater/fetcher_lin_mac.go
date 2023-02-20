//go:build linux || darwin
// +build linux darwin

package updater

import (
	"github.com/ActiveState/cli/pkg/unarchiver"
)

func blobUnarchiver(blob []byte) *unarchiver.TarGzBlob {
	return unarchiver.NewTarGzBlob(blob)
}

func checkAdmin() error {
	return nil
}
