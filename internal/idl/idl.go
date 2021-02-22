package idl

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
)

func UnixSocketFile(basePath string) string {
	return filepath.Join(basePath, constants.DaemonFile)
}

func NetworkPortFile(basePath string) string {
	return filepath.Join(basePath, constants.DaemonPortFile)
}
