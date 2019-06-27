package virtualenvironment

import (
	"os"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

var persisted *VirtualEnvironment

// FailAlreadyActive is a failure given when a project is already active
var FailAlreadyActive = failures.Type("virtualenvironment.fail.alreadyactive", failures.FailUser)

// OS is used by tests to spoof a different value
var OS = rt.GOOS

// VirtualEnvironment represents our virtual environment, it pulls together and virtualizes the runtime environment
type VirtualEnvironment struct {
	project             *project.Project
	onDownloadArtifacts func()
	onInstallArtifacts  func()
	artifactPaths       []string
}

// Get returns a persisted version of VirtualEnvironment{}
func Get() *VirtualEnvironment {
	if persisted == nil {
		persisted = Init()
	}
	return persisted
}

// Init creates an instance of VirtualEnvironment{} with default settings
func Init() *VirtualEnvironment {
	return &VirtualEnvironment{project: project.Get()}
}

// Activate the virtual environment
func (v *VirtualEnvironment) Activate() *failures.Failure {
	logging.Debug("Activating Virtual Environment")

	activeProject := os.Getenv(constants.ActivatedStateEnvVarName)
	if activeProject != "" {
		return FailAlreadyActive.New("err_already_active", v.project.Owner()+"/"+v.project.Name())
	}

	if OS != "darwin" && strings.ToLower(os.Getenv(constants.DisableRuntime)) != "true" {
		// Only Linux and Windows currently support runtime environments, but we still want to have virtual environments
		// on mac
		if failure := v.activateRuntime(); failure != nil {
			return failure
		}
	}

	return nil
}

// OnDownloadArtifacts will call the given function when artifacts are being downloaded
func (v *VirtualEnvironment) OnDownloadArtifacts(f func()) { v.onDownloadArtifacts = f }

// OnInstallArtifacts will call the given function when artifacts are being installed
func (v *VirtualEnvironment) OnInstallArtifacts(f func()) { v.onInstallArtifacts = f }

// activateRuntime sets up a runtime environment
func (v *VirtualEnvironment) activateRuntime() *failures.Failure {
	installer, fail := runtime.InitInstaller()
	if fail != nil {
		return fail
	}

	installer.OnDownload(v.onDownloadArtifacts)
	installer.OnInstall(v.onInstallArtifacts)
	if fail := installer.Install(); fail != nil {
		return fail
	}

	v.artifactPaths = installer.InstallDirs()

	return nil
}

// GetEnv returns a map of the cumulative environment variables for all active virtual environments
func (v *VirtualEnvironment) GetEnv() map[string]string {
	env := map[string]string{"PATH": os.Getenv("PATH")}

	for _, artifactPath := range v.artifactPaths {
		meta, fail := runtime.InitMetaData(artifactPath)
		if fail != nil {
			logging.Warning("Skipping Artifact '%s', could not retrieve metadata: %v", artifactPath, fail)
			continue
		}

		if meta.AffectedEnv != "" {
			env[meta.AffectedEnv] = ""
		}
		for _, v := range meta.BinaryLocations {
			path := v.Path
			if v.Relative {
				path = filepath.Join(artifactPath, path)
			}
			env["PATH"] = path + string(os.PathListSeparator) + env["PATH"]
		}

		// Add DLL dir to PATH on Windows
		if meta.RelocationTargetBinaries != "" && rt.GOOS == "windows" {
			env["PATH"] = filepath.Join(meta.Path, meta.RelocationTargetBinaries) + string(os.PathListSeparator) + env["PATH"]
		}
	}

	pjfile := projectfile.Get()
	env[constants.ActivatedStateEnvVarName] = filepath.Dir(pjfile.Path())

	if uid := os.Getenv(constants.ActivatedStateIDEnvVarName); uid != "" {
		env[constants.ActivatedStateIDEnvVarName] = uid
	}

	return env
}

// WorkingDirectory returns the working directory to use for the current environment
func (v *VirtualEnvironment) WorkingDirectory() string {
	wd, err := os.Getwd()
	if err != nil {
		// Shouldn't happen unless something is seriously wrong with your system
		panic(locale.T("panic_couldnt_detect_wd", map[string]interface{}{"Error": err.Error()}))
	}

	return wd
}
