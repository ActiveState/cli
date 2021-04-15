package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

func backupFiles(targetFiles []string) ([]string, error) {
	var renamed []string
	for _, t := range targetFiles {
		if fileutils.TargetExists(t) {
			newName := fmt.Sprintf("%s.bak", t)
			err := os.Rename(t, newName)
			if err != nil {
				// restore already renamed files and return with error
				_ = restoreFiles(renamed)
				return nil, errs.Wrap(err, "Failed to backup file %s", t)
			}
			renamed = append(renamed, newName)
		}
	}
	return renamed, nil
}

func restoreFiles(backupFiles []string) error {
	var errors []error
	for _, b := range backupFiles {
		origName := strings.TrimSuffix(b, ".bak")
		err := os.Rename(b, origName)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return errs.Wrap(errors[0], "Failed to restore some files.")
	}
	return nil
}

func removeBackupFiles(backupFiles []string) error {
	var errors []error
	for _, b := range backupFiles {
		err := os.Remove(b)
		if err != nil {
			errors = append(errors, err)
		}
	}
	if len(errors) > 0 {
		return errs.Wrap(errors[0], "Failed to remove some back-up files")
	}

	return nil
}

func Install(fromDir, toDir string) error {
	// Todo: https://www.pivotaltracker.com/story/show/177600107
	// Get target file paths.
	var targetFiles []string
	for _, file := range fileutils.ListDir(fromDir, false) {
		targetFile := filepath.Join(toDir, filepath.Base(file))
		targetFiles = append(targetFiles, targetFile)
	}
	logging.Debug("Target files=%s", strings.Join(targetFiles, ","))

	backups, err := backupFiles(targetFiles)
	if err != nil {
		return errs.Wrap(err, "Backup of existing files failed.")
	}
	defer removeBackupFiles(backups)

	// try to copy files to target directory
	if err := fileutils.CopyAndRenameFiles(fromDir, toDir); err != nil {
		// on failure ... restore back-up files (hopefully!!)
		restErr := restoreFiles(backups)
		if restErr != nil {
			logging.Error("restoring of backup files failed: %v", restErr)
		}
		logging.Debug("Successfully restored original files.")
		return errs.Wrap(err, "Failed to copy files to dir %s", toDir)
	}
	return nil
}
