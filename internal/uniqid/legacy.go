package uniqid

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
)

const legacyPersistDir = "activestate/persist"

func moveLegacyFile(destination string) error {
	legacyDir, err := legacyStorageDir()
	if err != nil {
		return errs.Wrap(err, "Could not get legacy storage directory")
	}

	// If the legacy file does not not exist there is nothing to move
	if !fileExists(filepath.Join(legacyDir, fileName)) {
		return nil
	}

	destinationDir := filepath.Dir(destination)
	err = mkdir(destinationDir)
	if err != nil {
		return errs.Wrap(err, "Could not create new persist directory")
	}

	err = moveAllFiles(legacyDir, destinationDir)
	if err != nil {
		return errs.Wrap(err, "Could not move legacy uniqid file")
	}

	// The legacy directory is a sub directory, we want to remove the parent
	err = os.RemoveAll(filepath.Dir(legacyDir))
	if err != nil {
		return errs.Wrap(err, "Could not remove legacy uniqid dir")
	}

	return nil
}

func legacyStorageDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errs.Wrap(err, "cannot get home dir for uniqid file")
	}

	return filepath.Join(home, "AppData", legacyPersistDir), nil
}
