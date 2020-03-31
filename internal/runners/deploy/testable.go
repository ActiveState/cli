package deploy

import (
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/runtime"
)

// Installable is an interface for runtime.Installer
type Installable interface {
	Install() (bool, *failures.Failure)
	InstallDirs() ([]string, *failures.Failure)
}

// NewInstallerFunc defines a testable type for runtime.InitInstaller
type NewInstallerFunc func(commitID strfmt.UUID, owner, projectName string, targetDir string) (Installable, *failures.Failure)

// NewInstaller wraps runtime.NewInstaller so we can modify the return types
func NewInstaller(commitID strfmt.UUID, owner, projectName, targetDir string) (Installable, *failures.Failure) {
	return runtime.NewInstallerByParams(runtime.NewInstallerParams(
		targetDir,
		commitID,
		owner,
		projectName,
	))
}

// DefaultBranchForProjectNameFunc defines a testable type for model.DefaultBranchForProjectName
type DefaultBranchForProjectNameFunc func(owner, name string) (*mono_models.Branch, *failures.Failure)
