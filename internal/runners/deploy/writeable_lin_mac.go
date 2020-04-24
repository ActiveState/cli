// +build !windows

package deploy

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/google/uuid"
)

func isWritable(path string) bool {
	// Check if we can write to this path
	fpath := filepath.Join(path, uuid.New().String())
	if err := fileutils.Touch(fpath); err != nil {
		logging.Error("Could not create file: %v", err)
		return false
	}

	if errr := os.Remove(fpath); errr != nil {
		logging.Error("Could not clean up test file: %v", errr)
		return false
	}

	return true
}
