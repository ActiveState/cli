// +build !darwin

package runtime

import (
	"runtime"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
)

// Prepare will assume the LibLocation in cases where the metadata
// doesn't contain it and we know what it should be
func (m *MetaData) Prepare() *failures.Failure {
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
	if m.hasBinaryFile(constants.ActivePython3Executable) || m.hasBinaryFile(constants.ActivePython2Executable) {
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
			var fail *failures.Failure
			if m.RelocationDir, fail = m.pythonRelocationDir(); fail != nil {
				return fail
			}
		}
		// Env
		m.setPythonEnv()

		//Perl
	} else if m.hasBinaryFile(constants.ActivePerlExecutable) {
		logging.Debug("Detected Perl artifact, ensuring backwards compatibility")

		// RelocationDir
		if m.RelocationDir == "" {
			var fail *failures.Failure
			if m.RelocationDir, fail = m.perlRelocationDir(); fail != nil {
				return fail
			}
		}
		// AffectedEnv
		if m.AffectedEnv == "" {
			m.AffectedEnv = "PERL5LIB"
		}
	} else {
		logging.Debug("No language detected for %s", m.Path)
	}

	if m.RelocationDir == "" {
		return FailMetaDataNotDetected.New("installer_err_runtime_missing_meta", m.Path)
	}

	return nil
}
