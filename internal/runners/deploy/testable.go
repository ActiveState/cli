package deploy

import (
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/failures"
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
type newInstallerFunc func(commitID, projectID strfmt.UUID, owner, projectName string, targetDir string) (installable, string, *failures.Failure)

// newInstaller wraps runtime.newInstaller so we can modify the return types
func newInstaller(commitID, projectID strfmt.UUID, owner, projectName, targetDir string) (installable, string, *failures.Failure) {
	params := runtime.NewInstallerParams(
		targetDir,
		commitID,
		projectID,
		owner,
		projectName,
	)
	installable, fail := runtime.NewInstallerByParams(params)
	return installable, params.CacheDir, fail
}
