package installer

import "github.com/ActiveState/cli/internal/failures"

var (
	// FailInvalidWorkingDir ...
	FailInvalidWorkingDir = failures.Type("installer.invalid.workingdir", failures.FailIO)

	// FailInvalidArchive ...
	FailInvalidArchive = failures.Type("installer.invalid.archive", failures.FailIO)
)

// Installer ...
type Installer interface {
	WorkingDir() string
	ArchivePath() string
	Install() *failures.Failure
}
