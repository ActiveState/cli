// +build linux windows

package installer

import (
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

func InstallSystemFiles(_, _, _ string) error {
	return nil
}

func (i *Installation) BackupFiles() error {
	// Todo: https://www.pivotaltracker.com/story/show/177600107
	// Get target file paths.
	var targetFiles []string
	for _, file := range fileutils.ListDir(i.fromDir, false) {
		targetFile := filepath.Join(i.binaryDir, filepath.Base(file))
		targetFiles = append(targetFiles, targetFile)
	}
	logging.Debug("Target files=%s", strings.Join(targetFiles, ","))

	backups, err := backupFiles(targetFiles)
	if err != nil {
		return errs.Wrap(err, "Backup of existing files failed.")
	}
	i.backups = append(i.backups, backups...)
	return nil
}
