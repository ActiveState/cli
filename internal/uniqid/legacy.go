package uniqid

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
)

const legacyPersistDir = "activestate/persist"

func moveUniqidFile(destination string) error {
	legacyDir, err := legacyStorageDir()
	if err != nil {
		return errs.Wrap(err, "Could not get legacy storage directory")
	}

	legacyUniqIDFile := filepath.Join(legacyDir, fileName)

	// If the uniqID file does not not exist there is nothing to move
	if !fileExists(legacyUniqIDFile) {
		return nil
	}

	err = mkdirUnlessExists(filepath.Dir(destination))
	if err != nil {
		return errs.Wrap(err, "Could not create new persist directory")
	}

	err = copyFile(legacyUniqIDFile, destination)
	if err != nil {
		return errs.Wrap(err, "Could not move legacy uniqid file")
	}

	// Ignore removal errors that could occur due to permissions issues
	// Remove the legacy uniqid file and its parent directories
	// The legacy directory is a sub directory, we want to remove the parent
	_ = os.Remove(legacyUniqIDFile)
	_ = os.Remove(legacyDir)
	_ = os.Remove(filepath.Dir(legacyDir))

	return nil
}

func legacyStorageDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errs.Wrap(err, "cannot get home dir for uniqid file")
	}

	return filepath.Join(home, "AppData", legacyPersistDir), nil
}
