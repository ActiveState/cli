package runtime

import (
	"path/filepath"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/hash"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

type Runtime struct {
	runtimeDir  string
	commitID    strfmt.UUID
	owner       string
	projectName string
	msgHandler  MessageHandler
}

func NewRuntime(projectDir string, commitID strfmt.UUID, owner string, projectName string, msgHandler MessageHandler) (*Runtime, error) {
	var resolvedProjectDir string
	if projectDir != "" {
		var err error
		projectDir = strings.TrimSuffix(projectDir, constants.ConfigFileName)
		resolvedProjectDir, err = fileutils.ResolveUniquePath(projectDir)
		if err != nil {
			return nil, locale.WrapError(err, "err_new_runtime_unique_path", "Failed to resolve a unique file path to the project dir.")
		}
		logging.Debug("In NewRuntime: resolved project dir is: %s", resolvedProjectDir)
	}

	installPath := filepath.Join(config.CachePath(), hash.ShortHash(resolvedProjectDir))
	return &Runtime{installPath, commitID, owner, projectName, msgHandler}, nil
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
