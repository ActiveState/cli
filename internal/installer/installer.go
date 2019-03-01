package installer

import "github.com/ActiveState/cli/internal/failures"

var (
	// FailInstallDirInvalid represents a Failure due to the working-dir for an installation being invalid in some way.
	FailInstallDirInvalid = failures.Type("installer.installdir.invalid", failures.FailIO)

	// FailArchiveInvalid represents a Failure due to the installer archive file being invalid in some way.
	FailArchiveInvalid = failures.Type("installer.archive.invalid", failures.FailIO)

	// FailDistInvalid represents a Failure due to a distribution being invalid in some way prior to installation.
	FailDistInvalid = failures.Type("installer.dist.invalid", failures.FailIO)

	// FailDistInstallation represents a Failure to install a distribution.
	FailDistInstallation = failures.Type("installer.dist.installation", failures.FailOS)
)

// Installer defines the functionality for implementations of distribution installers.
type Installer interface {
	// DistributionName returns a qualified name of a distribution to be installed.
	DistributionName() string

	// InstallDir is the base directory where a distribution will be installed to.
	InstallDir() string

	// ArchivePath is the path to an installer's archive file.
	ArchivePath() string

	// Install will perform the actual installation of a distribution.
	Install() *failures.Failure
}
