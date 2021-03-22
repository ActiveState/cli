package runtime

import (
	"path/filepath"
	"strings"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
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
	installer   *Installer
}

func NewRuntime(projectDir, cachePath string, commitID strfmt.UUID, owner string, projectName string, msgHandler MessageHandler) (*Runtime, error) {
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

	installPath := filepath.Join(cachePath, hash.ShortHash(resolvedProjectDir))
	r := &Runtime{
		runtimeDir:  installPath,
		commitID:    commitID,
		owner:       owner,
		projectName: projectName,
		msgHandler:  msgHandler,
	}

	r.installer = NewInstaller(r)

	analytics.Event(catRuntime, actStart)
	if r.IsCachedRuntime() {
		analytics.Event(catRuntime, actCache)
	}

	return r, nil
}

func (r *Runtime) SetInstallPath(installPath string) {
	logging.Debug("SetInstallPath: %s", installPath)
	r.runtimeDir = installPath
}

func (r *Runtime) InstallPath() string {
	return r.runtimeDir
}

// IsCachedRuntime checks if the requested runtime is already available ie.,
// the runtime installation completed successful (marker file found) AND the requested commitID did not change
func (r *Runtime) IsCachedRuntime() bool {
	marker := filepath.Join(r.runtimeDir, constants.RuntimeInstallationCompleteMarker)
	if !fileutils.FileExists(marker) {
		logging.Debug("Marker does not exist: %s", marker)
		return false
	}

	contents, err := fileutils.ReadFile(marker)
	if err != nil {
		logging.Error("Could not read marker: %v", err)
		return false
	}

	logging.Debug("IsCachedRuntime for %s, %s==%s", marker, string(contents), r.commitID.String())
	return string(contents) == r.commitID.String()
}

// MarkInstallationComplete writes the installation complete marker to the runtime directory
func (r *Runtime) MarkInstallationComplete() error {
	markerFile := filepath.Join(r.runtimeDir, constants.RuntimeInstallationCompleteMarker)
	markerDir := filepath.Dir(markerFile)
	err := fileutils.MkdirUnlessExists(markerDir)
	if err != nil {
		return errs.Wrap(err, "could not create completion marker directory")
	}
	err = fileutils.WriteFile(markerFile, []byte(r.commitID.String()))
	if err != nil {
		return errs.Wrap(err, "could not set completion marker")
	}
	return nil
}

// StoreBuildEngine stores the build engine value in the runtime directory
func (r *Runtime) StoreBuildEngine(buildEngine BuildEngine) error {
	storeFile := filepath.Join(r.runtimeDir, constants.RuntimeBuildEngineStore)
	storeDir := filepath.Dir(storeFile)
	logging.Debug("Storing build engine %s at %s", buildEngine.String(), storeFile)
	err := fileutils.MkdirUnlessExists(storeDir)
	if err != nil {
		return errs.Wrap(err, "Could not create completion marker directory.")
	}
	err = fileutils.WriteFile(storeFile, []byte(buildEngine.String()))
	if err != nil {
		return errs.Wrap(err, "Could not store build engine string.")
	}
	return nil
}

// BuildEngine returns the runtime build engine value stored in the runtime directory
func (r *Runtime) BuildEngine() (BuildEngine, error) {
	storeFile := filepath.Join(r.runtimeDir, constants.RuntimeBuildEngineStore)

	data, err := fileutils.ReadFile(storeFile)
	if err != nil {
		return UnknownEngine, errs.Wrap(err, "Could not read build engine cache store.")
	}

	return parseBuildEngine(string(data)), nil
}

// Env will grab the environment information for the given runtime.
// This currently just aliases to installer, pending further refactoring
func (r *Runtime) Env() (EnvGetter, error) {
	return r.installer.Env()
}

func (r *Runtime) CommitID() strfmt.UUID {
	return r.commitID
}

func (r *Runtime) Owner() string {
	return r.owner
}

func (r *Runtime) ProjectName() string {
	return r.projectName
}
