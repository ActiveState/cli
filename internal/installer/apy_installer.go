package installer

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/mholt/archiver"
)

// ActivePythonInstaller ...
type ActivePythonInstaller struct {
	workingDir  string
	archivePath string
}

// NewActivePythonInstaller ...
// - workingDir represents the working dir; e.g. a virtualenvironment directory
// - archivePath is the path to a tarball (in this case)
func NewActivePythonInstaller(workingDir, archivePath string) (*ActivePythonInstaller, *failures.Failure) {
	if !fileutils.DirExists(workingDir) {
		return nil, FailInvalidWorkingDir.New("installer_err_invalid_workingdir", workingDir)
	} else if !fileutils.FileExists(archivePath) {
		return nil, FailInvalidArchive.New("installer_err_notfound_archive", archivePath)
	} else if !archiver.TarGz.Match(archivePath) {
		return nil, FailInvalidArchive.New("installer_err_badext_archive", archivePath)
	}
	return &ActivePythonInstaller{
		workingDir:  workingDir,
		archivePath: archivePath,
	}, nil
}

// WorkingDir ...
func (installer *ActivePythonInstaller) WorkingDir() string {
	return installer.workingDir
}

// ArchivePath ...
func (installer *ActivePythonInstaller) ArchivePath() string {
	return installer.archivePath
}

// Install ...
func (installer *ActivePythonInstaller) Install() *failures.Failure {
	return nil
}
