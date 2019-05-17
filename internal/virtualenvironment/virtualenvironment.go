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
	"github.com/ActiveState/cli/internal/shimming"
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
	env                 map[string]string
	envPath             []string
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
	return &VirtualEnvironment{
		project: project.Get(),
		env:     map[string]string{},
		envPath: []string{},
	}
}

// Activate the virtual environment
func (v *VirtualEnvironment) Activate() *failures.Failure {
	logging.Debug("Activating Virtual Environment")

	activeProject := os.Getenv(constants.ActivatedStateEnvVarName)
	if activeProject != "" {
		return FailAlreadyActive.New("err_already_active", v.project.Owner()+"/"+v.project.Name())
	}

	// expand project vars to environment vars
	for _, variable := range v.project.Variables() {
		val, fail := variable.Value()
		if fail != nil {
			return fail
		}
		if variable.Name() == "PATH" {
			v.envPath = append(v.envPath, variable.Name())
		} else {
			v.env[variable.Name()] = val
		}
	}

	if OS == "linux" && strings.ToLower(os.Getenv(constants.DisableRuntime)) != "true" {
		// Only Linux currently supports runtime environments, but we still want to have virtual environments
		// on other platforms
		if fail := v.activateRuntime(); fail != nil {
			return fail
		}

		if fail := v.activateShims(v.BinaryPaths(), shimming.InitCollection()); fail != nil {
			return fail
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

func (v *VirtualEnvironment) activateShims(binaryPaths []string, collection *shimming.Collection) *failures.Failure {
	binaryPath, fail := collection.ShimBinaries(binaryPaths)
	if fail != nil {
		return fail
	}

	if binaryPath != "" {
		v.envPath = append(v.envPath, binaryPath)
	}

	return nil
}

func (v *VirtualEnvironment) BinaryPaths() []string {
	binaryPaths := []string{}
	for _, artifactPath := range v.artifactPaths {
		meta, fail := v.metaForArtifact(artifactPath)
		if fail != nil || meta == nil {
			logging.Warning("Skipping Artifact binaries '%s', could not retrieve metadata: %v", artifactPath, fail)
			continue
		}

		for _, v := range meta.BinaryLocations {
			path := v.Path
			if v.Relative {
				path = filepath.Join(artifactPath, path)
			}
			binaryPaths = append(binaryPaths, path)
		}
	}

	return binaryPaths
}

// GetEnv returns a map of the cumulative environment variables for all active virtual environments
func (v *VirtualEnvironment) GetEnv() map[string]string {
	env := v.env
	env["PATH"] = ""

	for _, path := range v.BinaryPaths() {
		env["PATH"] = path + string(os.PathListSeparator) + env["PATH"]
	}
	for _, artifactPath := range v.artifactPaths {
		meta, fail := v.metaForArtifact(artifactPath)
		if fail != nil {
			logging.Warning("Skipping Artifact affectedenv '%s', could not retrieve metadata: %v", artifactPath, fail)
			continue
		}

		if meta.AffectedEnv != "" {
			env[meta.AffectedEnv] = ""
		}
	}

	// Prepend PATH values
	env["PATH"] = strings.Join(v.envPath, string(os.PathListSeparator)) + string(os.PathListSeparator) + env["PATH"]

	// Append the original PATH
	env["PATH"] = env["PATH"] + string(os.PathListSeparator) + os.Getenv("PATH")

	pjfile := projectfile.Get()
	env[constants.ActivatedStateEnvVarName] = filepath.Dir(pjfile.Path())

	return env
}

func (v *VirtualEnvironment) metaForArtifact(artifactPath string) (*runtime.MetaData, *failures.Failure) {
	meta, fail := runtime.InitMetaData(artifactPath)
	if fail == nil {
		return meta, nil
	}

	if !fail.Type.Matches(runtime.FailMetaDataNotFound) {
		return meta, fail
	}

	// If no meta file can be found we instead assume a bin directory. This is to facilitate legacy builds
	logging.Debug("Artifact '%s' has no metadata file, assuming it has a bin directory: %v", artifactPath, fail)
	return &runtime.MetaData{
		BinaryLocations: []runtime.MetaDataBinary{
			runtime.MetaDataBinary{
				Path:     "bin",
				Relative: true,
			},
		},
	}, nil
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
