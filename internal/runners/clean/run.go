package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/runners/prepare"
)

func removeCache(cachePath string) error {
	err := os.RemoveAll(cachePath)
	if err != nil {
		return locale.WrapError(err, "err_remove_cache", "Could not remove State Tool cache directory")
	}
	return nil
}

func undoPrepare(cfg configurable) error {
	toRemove := prepare.InstalledPreparedFiles(cfg)

	var aggErr error
	for _, f := range toRemove {
		if fileutils.TargetExists(f) {
			err := os.RemoveAll(f)
			if err != nil {
				aggErr = locale.WrapError(aggErr, "err_undo_prepare_remove_file", "Failed to remove file {{.V0}}", f)
			}
		}
	}

	return aggErr
}
