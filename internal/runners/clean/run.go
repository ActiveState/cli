package clean

import (
	"os"

	"github.com/ActiveState/cli/internal/locale"
)

func removeCache(cachePath string) error {
	err := os.RemoveAll(cachePath)
	if err != nil {
		return locale.WrapError(err, "err_remove_cache", "Could not remove State Tool cache directory")
	}
	return nil
}
