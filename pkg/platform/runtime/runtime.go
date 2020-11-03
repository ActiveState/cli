package runtime

import (
	"path/filepath"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
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

func (r *Runtime) IsCachedRuntime() bool {
	marker := filepath.Join(r.runtimeDir, constants.RuntimeInstallationCompleteMarker)
	if !fileutils.FileExists(marker) {
		return false
	}

	contents, fail := fileutils.ReadFile(marker)
	if fail != nil {
		logging.Error("Could not read marker: %v", fail)
		return false
	}

	return string(contents) == r.commitID.String()
}

func (r *Runtime) MarkInstallationComplete() error {
	markerFile := filepath.Join(r.runtimeDir, constants.RuntimeInstallationCompleteMarker)
	markerDir := filepath.Base(markerFile)
	fail := fileutils.MkdirUnlessExists(markerDir)
	if fail != nil {
		return errs.Wrap(fail, "could not create completion marker directory")
	}
	fail = fileutils.WriteFile(markerFile, []byte(r.commitID.String()))
	if fail != nil {
		return errs.Wrap(fail, "could not set completion marker")
	}
	return nil
}

func (r *Runtime) StoreBuildEngine(buildEngine BuildEngine) error {
	storeFile := filepath.Join(r.runtimeDir, constants.RuntimeBuildEngineStore)
	storeDir := filepath.Base(storeFile)
	fail := fileutils.MkdirUnlessExists(storeDir)
	if fail != nil {
		return errs.Wrap(fail, "Could not create completion marker directory.")
	}
	fail = fileutils.WriteFile(storeFile, []byte(buildEngine.String()))
	if fail != nil {
		return errs.Wrap(fail, "Could not store build engine string.")
	}
	return nil
}

func (r *Runtime) BuildEngine() (BuildEngine, error) {
	storeFile := filepath.Join(r.runtimeDir, constants.RuntimeBuildEngineStore)

	data, fail := fileutils.ReadFile(storeFile)
	if fail != nil {
		return UnknownEngine, errs.Wrap(fail, "Could not read build engine cache store.")
	}

	return parseBuildEngine(string(data)), nil
}

// Env will grab the environment information for the given runtime.
// This currently just aliases to installer, pending further refactoring
func (r *Runtime) Env() (EnvGetter, *failures.Failure) {
	return NewInstaller(r).Env()
}

func (r *Runtime) ArtifactsFromCache() ([]*HeadChefArtifact, BuildEngine, error) {
	return []*HeadChefArtifact{}, UnknownEngine, nil
}
