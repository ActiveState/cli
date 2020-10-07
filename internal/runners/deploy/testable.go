package deploy

import (
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/pkg/platform/api/mono/mono_models"
	"github.com/ActiveState/cli/pkg/platform/runtime"
)

type envGetter = runtime.EnvGetter

// installable is an interface for runtime.Installer
type installable interface {
	Install() (envGetter envGetter, freshInstallation bool, fail *failures.Failure)
	Env() (envGetter envGetter, fail *failures.Failure)
	IsInstalled() (bool, *failures.Failure)
}

// newInstallerFunc defines a testable type for runtime.InitInstaller
type newInstallerFunc func(rt *runtime.Runtime) installable

// newInstaller wraps runtime.newInstaller so we can modify the return types
func newInstaller(rt *runtime.Runtime) installable {
	installable := runtime.NewInstaller(rt)
	return installable
}

// defaultBranchForProjectNameFunc defines a testable type for model.DefaultBranchForProjectName
type defaultBranchForProjectNameFunc func(owner, name string) (*mono_models.Branch, *failures.Failure)
