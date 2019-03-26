package runtime

import (
	"os"

	"github.com/ActiveState/archiver"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

var (
	// FailInstallDirInvalid represents a Failure due to the working-dir for an installation being invalid in some way.
	FailInstallDirInvalid = failures.Type("runtime.installdir.invalid", failures.FailIO)

	// FailArchiveInvalid represents a Failure due to the installer archive file being invalid in some way.
	FailArchiveInvalid = failures.Type("runtime.archive.invalid", failures.FailIO)

	// FailRuntimeInvalid represents a Failure due to a runtime being invalid in some way prior to installation.
	FailRuntimeInvalid = failures.Type("runtime.runtime.invalid", failures.FailIO)

	// FailRuntimeInstallation represents a Failure to install a runtime.
	FailRuntimeInstallation = failures.Type("runtime.runtime.installation", failures.FailOS)
)

// Installer defines the functionality for implementations of runtime installers. Runtimes are dependent
// on there existing an archive that they are distributed within.
type Installer interface {
	// InstallDir is the base directory where a runtime will be installed to.
	InstallDir() string

	// Install will download the installer and perform the install
	Install() *failures.Failure

	// InstallFromArchive will perform the actual installation of a runtime given an installer archive.
	InstallFromArchive(archivePath string) *failures.Failure

	// OnDownload is triggered when the installer is being downloaded
	OnDownload(func())

	// OnInstall is triggered when the installer is being ran (ie. being installed)
	OnInstall(func())
}

// RuntimeInstaller implements an Installer that works with a runtime.Downloader and a
// runtime.Installer. Effectively, upon calling Install, the RuntimeInstaller will first
// try and Download an archive, then it will try to install that downloaded archive.
type RuntimeInstaller struct {
	runtimeDownloader Downloader
	onDownload        func()
	onInstall         func()
}

// validateArchiveTarGz ensures the given path to archive is an actual file and that its suffix is a well-known
// suffix for tar+gz files.
func validateArchiveTarGz(archivePath string) *failures.Failure {
	if !fileutils.FileExists(archivePath) {
		return FailArchiveInvalid.New("installer_err_archive_notfound", archivePath)
	} else if archiver.DefaultTarGz.CheckExt(archivePath) != nil {
		return FailArchiveInvalid.New("installer_err_archive_badext", archivePath)
	}
	return nil
}

// removeInstallDir will remove a given directory and log any errors resulting from
// that removal. No errors are returned.
func removeInstallDir(dir string) {
	if err := os.RemoveAll(dir); err != nil {
		logging.Errorf("attempting to remove install dir '%s': %v", dir, err)
	}
}
