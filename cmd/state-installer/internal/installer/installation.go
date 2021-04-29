package installer

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

type Installation struct {
	fromDir string
	toDir   string
	backups []string
}

func backupFiles(targetFiles []string) ([]string, error) {
	var renamed []string
	for _, t := range targetFiles {
		if fileutils.TargetExists(t) {
			// Note: We use the unconventional suffix .bac to support transitional updates on Windows, as the following can happen:
			//   - Legacy State Tool is invoked eg., as `state.exe update`
			//   - The transitional executable is pulled down and invoked as `state.exe _prepare` (invoking state tool is now called `state.exe.bak`)
			//   - The installer cannot rename the transitional `state.exe` to `state.exe.bak` (but to `state.exe.bac`)
			// Phew!
			newName := fmt.Sprintf("%s.bac", t)
			if fileutils.TargetExists(newName) {
				_ = os.Remove(newName)
			}
			if err := os.Rename(t, newName); err != nil {
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
		origName := strings.TrimSuffix(b, ".bac")
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

func New(fromDir, toDir string) *Installation {
	return &Installation{
		fromDir, toDir, nil,
	}
}

func (i *Installation) RemoveBackupFiles() error {
	var es []error
	for _, b := range i.backups {
		err := os.Remove(b)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			// On Windows, if the executable was still running, the removal of the backup could fail here.
			// We are trying to hide the file such that a .bac file does not (visually!) litter the folder.
			errHide := fileutils.HideFile(b)
			if errHide != nil {
				logging.Error("Encountered error hiding file %s: %v", b, err)
			}
			es = append(es, err)
		}
	}
	if len(es) > 0 {
		return errs.Wrap(es[0], "Failed to remove some back-up files")
	}

	return nil
}

func (i *Installation) BackupFiles() error {
	// Todo: https://www.pivotaltracker.com/story/show/177600107
	// Get target file paths.
	var targetFiles []string
	for _, file := range fileutils.ListDir(i.fromDir, false) {
		targetFile := filepath.Join(i.toDir, filepath.Base(file))
		targetFiles = append(targetFiles, targetFile)
	}
	logging.Debug("Target files=%s", strings.Join(targetFiles, ","))

	backups, err := backupFiles(targetFiles)
	if err != nil {
		return errs.Wrap(err, "Backup of existing files failed.")
	}
	i.backups = backups
	return nil
}

func (i *Installation) Rollback() error {
	return restoreFiles(i.backups)
}

func (i *Installation) Install() error {
	if err := i.BackupFiles(); err != nil {
		return errs.Wrap(err, "Failed to backup original files.")
	}
	if err := fileutils.CopyAndRenameFiles(i.fromDir, i.toDir); err != nil {
		return errs.Wrap(err, "Failed to copy installation files to dir %s", i.toDir)
	}
	if err := InstallSystemFiles(i.toDir); err != nil {
		return errs.Wrap(err, "Installation of system files failed.")
	}

	return nil
}
