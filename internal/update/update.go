package update

import (
	"os"

	updater "github.com/inconshreveable/go-update"
)

// Update updates the ActiveState-CLI executable with the one pointed to by the
// given path. That executable is assumed to be trusted (checksum verified,
// public key signature confirmed, etc.).
func Update(exePath string) error {
	file, err := os.Open(exePath)
	if err != nil {
		return err
	}
	defer file.Close()
	return updater.Apply(file, updater.Options{})
}
