package envdef

import (
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

// Constants is a map of constants that are being expanded in environment variables and file transformations to their installation-specific values
type Constants map[string]string

// NewConstants initializes a new map of constants that will need to be set to installation-specific values
// Currently it only has one field `INSTALLDIR`
func NewConstants(installdir string) Constants {
	return map[string]string{
		`INSTALLDIR`: caseSensitiveInstallDir(installdir),
	}
}

func caseSensitiveInstallDir(installDir string) string {
	caseSensitiveInstallDir, err := fileutils.CaseSensitivePath(installDir)
	if err != nil {
		logging.Error("Could not search for case sensitive install dir, error: %v", err)
		return installDir
	}
	return caseSensitiveInstallDir
}
