package runtime

import (
	"path/filepath"
	"runtime"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/hash"
	"github.com/ActiveState/cli/internal/logging"
)

type Runtime struct {
	runtimeDir  string
	commitID    strfmt.UUID
	owner       string
	projectName string
	msgHandler  MessageHandler
}

func NewRuntime(commitID strfmt.UUID, owner string, projectName string, msgHandler MessageHandler) *Runtime {
	var installPath string
	if runtime.GOOS == "darwin" {
		// mac doesn't use relocation so we can safely use a longer path
		installPath = filepath.Join(config.CachePath(), owner, projectName)
	} else {
		installPath = filepath.Join(config.CachePath(), hash.ShortHash(owner, projectName))
	}
	return &Runtime{installPath, commitID, owner, projectName, msgHandler}
}

func (r *Runtime) SetInstallPath(installPath string) {
	logging.Debug("SetInstallPath: %s", installPath)
	r.runtimeDir = installPath
}

func (r *Runtime) InstallPath() string {
	return r.runtimeDir
}

// Env will grab the environment information for the given runtime.
// This currently just aliases to installer, pending further refactoring
func (r *Runtime) Env() (EnvGetter, *failures.Failure) {
	return NewInstaller(r).Env()
}
