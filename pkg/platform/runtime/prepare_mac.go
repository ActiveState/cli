// +build darwin

package runtime

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

// Prepare ensures Metadata can handle Python runtimes on MacOS.
// These runtimes do not include metadata files as they should
// be runnable from where they are unarchived
func (m *MetaData) Prepare() *failures.Failure {
	m.BinaryLocations = []MetaDataBinary{
		MetaDataBinary{
			Path:     "Library/Frameworks/Python.framework/Versions/Current/bin/",
			Relative: true,
		},
	}

	if !m.hasBinaryFile(constants.ActivePython3Executable) && !m.hasBinaryFile(constants.ActivePython2Executable) {
		logging.Debug("No language detected for %s", m.Path)
		return nil
	}

	m.setPythonEnv()

	libDir := filepath.Join(m.Path, "Library/Frameworks/Python.framework/Versions/Current/lib")
	dirRe := regexp.MustCompile(`python\d.\d`)

	files, err := ioutil.ReadDir(libDir)
	if err != nil {
		return failures.FailOS.Wrap(err)
	}

	var sitePackages string
	for _, f := range files {
		if !f.IsDir() {
			continue
		}
		if dirRe.MatchString(f.Name()) {
			sitePackages = filepath.Join(libDir, f.Name(), "site-packages")
			break
		}
	}

	if fileutils.DirExists(sitePackages) {
		m.Env["PYTHONPATH"] = m.Env["PYTHONPATH"] + string(os.PathListSeparator) + sitePackages
	}

	return nil
}
