// +build !darwin

package camel

import (
	"runtime"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

// Prepare will assume the LibLocation in cases where the metadata
// doesn't contain it and we know what it should be
func (m *MetaData) Prepare(installRoot string) error {
	// BinaryLocations
	if m.BinaryLocations == nil || len(m.BinaryLocations) == 0 {
		m.BinaryLocations = []MetaDataBinary{
			MetaDataBinary{
				Path:     "bin",
				Relative: true,
			},
		}
	}

	// Python
	if m.hasBinaryFile(installRoot, constants.ActivePython3Executable) || m.hasBinaryFile(installRoot, constants.ActivePython2Executable) {
		logging.Debug("Detected Python artifact, ensuring backwards compatibility")

		// RelocationTargetBinaries
		if m.RelocationTargetBinaries == "" {
			if runtime.GOOS == "windows" {
				m.RelocationTargetBinaries = "DLLs"
			} else {
				m.RelocationTargetBinaries = "lib"
			}
		}
		// RelocationDir
		if m.RelocationDir == "" {
			var err error
			if m.RelocationDir, err = m.pythonRelocationDir(installRoot); err != nil {
				return err
			}
		}
		// Env
		m.setPythonEnv()

		//Perl
	} else if m.hasBinaryFile(installRoot, constants.ActivePerlExecutable) {
		logging.Debug("Detected Perl artifact, ensuring backwards compatibility")

		// RelocationDir
		if m.RelocationDir == "" {
			var err error
			if m.RelocationDir, err = m.perlRelocationDir(installRoot); err != nil {
				return err
			}
		}
		// AffectedEnv
		if m.AffectedEnv == "" {
			m.AffectedEnv = "PERL5LIB"
		}
	} else {
		logging.Debug("No language detected for %s", installRoot)
	}

	if m.RelocationDir == "" {
		return locale.NewError("installer_err_runtime_missing_meta", "", installRoot)
	}

	return nil
}
