package python

import (
	"io"
	"os"
	"path"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

// VirtualEnvironment covers the virtualenvironment.VirtualEnvironment interface, reference that for documentation
type VirtualEnvironment struct {
	datadir string
}

// NewVirtualEnvironment returns a configured python virtualenvironment.
func NewVirtualEnvironment(datadir string) *VirtualEnvironment {
	return &VirtualEnvironment{
		datadir: datadir,
	}
}

// Language - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Language() string {
	return "python3"
}

// DataDir - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) DataDir() string {
	return v.datadir
}

// WorkingDirectory - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) WorkingDirectory() string {
	return ""
}

// Activate - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Activate() *failures.Failure {
	if err := fileutils.Mkdir(v.datadir, "bin"); err != nil {
		return err
	}
	return fileutils.Mkdir(v.datadir, "lib")
}

// Env - see virtualenvironment.VirtualEnvironment
func (v *VirtualEnvironment) Env() map[string]string {
	env := map[string]string{}
	if distPath, found := v.pathToAnyDistribution(); found {
		logging.Debug("found distribution '%s'", distPath)
		env["PATH"] = path.Join(distPath, "bin")
	}
	return env
}

// pathToAnyDistribution will return the path to the first distribution dir found.
func (v *VirtualEnvironment) pathToAnyDistribution() (string, bool) {
	distsDirPath := path.Join(v.datadir, constants.ActivePythonDistsDir)
	if !fileutils.DirExists(distsDirPath) {
		logging.Debug("distributions dir '%s' does not exist", distsDirPath)
		return "", false
	}

	distsDir, err := os.Open(distsDirPath)
	if err != nil {
		logging.Error("accessing distributions dir '%s': %v", distsDirPath, err)
		return "", false
	}
	defer distsDir.Close()

	// read one directory name
	distDirNames, err := distsDir.Readdirnames(1)
	if err != nil {
		if err == io.EOF {
			logging.Debug("no distributions found in '%s'", distsDirPath)
		} else {
			logging.Error("reading dir names from distributions dir '%s': %v", distsDirPath, err)
		}
		return "", false
	}

	return path.Join(distsDirPath, distDirNames[0]), true
}
