// +build !darwin

package runtime

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

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

		// On Linux we must set the PERL5LIB in order for the modules to
		// be useable
		if runtime.GOOS == "linux" {
			// Currently only Perl 5.x.x is supported by the platform
			lib := filepath.Join(m.Path, "lib", "perl5")
			sitePerl := filepath.Join(m.Path, "lib", "perl5", "site_perl")

			m.Env["PERL5LIB"] = strings.Join([]string{m.Env["PERL5LIB"], lib, sitePerl}, string(os.PathListSeparator))
		}
	} else {
		logging.Debug("No language detected for %s", m.Path)
	}

	if m.RelocationDir == "" {
		return FailMetaDataNotDetected.New("installer_err_runtime_missing_meta")
	}

	return nil
}
