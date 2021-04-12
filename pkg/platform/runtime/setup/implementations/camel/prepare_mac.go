// +build darwin

package camel

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

// Prepare ensures Metadata can handle Python runtimes on MacOS.
// These runtimes do not include metadata files as they should
// be runnable from where they are unarchived
func (m *MetaData) Prepare(installRoot string) error {
	frameWorkDir := "Library/Frameworks/Python.framework/Versions/"
	m.BinaryLocations = []MetaDataBinary{
		MetaDataBinary{
			Path:     filepath.Join(frameWorkDir, "Current", "bin"),
			Relative: true,
		},
	}

	if !m.hasBinaryFile(installRoot, constants.ActivePython3Executable) && !m.hasBinaryFile(installRoot, constants.ActivePython2Executable) {
		logging.Debug("No language detected for %s", installRoot)
		return nil
	}

	m.setPythonEnv()

	libDir := filepath.Join(installRoot, frameWorkDir, "Current", "lib")
	dirRe := regexp.MustCompile(`python\d+.\d+`)

	files, err := ioutil.ReadDir(libDir)
	if err != nil {
		return errs.Wrap(err, "OS failure")
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

	if m.TargetedRelocations == nil {
		// the binaries are actually in a versioned directory
		// this version is likely the same as the found above, but it doesn't hurt to get explicitly
		dirRe = regexp.MustCompile(`\d+(?:\.\d+)+`)
		files, err = ioutil.ReadDir(filepath.Join(installRoot, frameWorkDir))
		if err != nil {
			return errs.Wrap(err, "OS failure")
		}

		var relVersionedFrameWorkDir string
		for _, f := range files {
			if !f.IsDir() {
				continue
			}
			if dirRe.MatchString(f.Name()) {
				relVersionedFrameWorkDir = filepath.Join(frameWorkDir, f.Name())
				break
			}
		}

		if relVersionedFrameWorkDir == "" {
			return errs.New("could not find path %s/x.x in build artifact", frameWorkDir)
		}

		m.TargetedRelocations = []TargetedRelocation{TargetedRelocation{
			InDir:        filepath.Join(frameWorkDir, "Current", "bin"),
			SearchString: "#!" + filepath.Join("/", relVersionedFrameWorkDir),
			Replacement:  "#!" + filepath.Join("${INSTALLDIR}", relVersionedFrameWorkDir),
		}}
	}

	return nil
}
