package virtualenvironment

import (
	"fmt"
	"os"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/google/uuid"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

var persisted *VirtualEnvironment

// FailAlreadyActive is a failure given when a project is already active
var FailAlreadyActive = failures.Type("virtualenvironment.fail.alreadyactive", failures.FailUser)

// OS is used by tests to spoof a different value
var OS = rt.GOOS

type getEnvFunc func(inherit bool, projectDir string) (map[string]string, *failures.Failure)

// VirtualEnvironment represents our virtual environment, it pulls together and virtualizes the runtime environment
type VirtualEnvironment struct {
	project             *project.Project
	activationID        string
	onDownloadArtifacts func()
	onInstallArtifacts  func()
	onUseCache          func()
	getEnv              getEnvFunc
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
		project:      project.Get(),
		activationID: uuid.New().String(),
	}
}

func New(getEnv getEnvFunc) *VirtualEnvironment {
	return &VirtualEnvironment{
		activationID: uuid.New().String(),
		getEnv:       getEnv,
	}
}

// Activate the virtual environment
func (v *VirtualEnvironment) Activate() *failures.Failure {
	logging.Debug("Activating Virtual Environment")

	activeProject := os.Getenv(constants.ActivatedStateEnvVarName)
	if activeProject != "" {
		return FailAlreadyActive.New("err_already_active", v.project.Owner()+"/"+v.project.Name())
	}

	if strings.ToLower(os.Getenv(constants.DisableRuntime)) != "true" {
		if failure := v.activateRuntime(); failure != nil {
			return failure
		}
	} else {
		fmt.Println("Skipping runtime activation")
	}

	return nil
}

// OnDownloadArtifacts will call the given function when artifacts are being downloaded
func (v *VirtualEnvironment) OnDownloadArtifacts(f func()) { v.onDownloadArtifacts = f }

// OnInstallArtifacts will call the given function when artifacts are being installed
func (v *VirtualEnvironment) OnInstallArtifacts(f func()) { v.onInstallArtifacts = f }

// OnUseCache will call the given function when the cached runtime is used
func (v *VirtualEnvironment) OnUseCache(f func()) { v.onUseCache = f }

// activateRuntime sets up a runtime environment
func (v *VirtualEnvironment) activateRuntime() *failures.Failure {
	pj := project.Get()
	installer, fail := runtime.NewInstaller(pj.CommitUUID(), pj.Owner(), pj.Name())
	if fail != nil {
		return fail
	}

	installer.OnDownload(v.onDownloadArtifacts)

	rt, installed, fail := installer.Install()
	if fail != nil {
		return fail
	}

	v.getEnv = rt.GetEnv
	if !installed && v.onUseCache != nil {
		v.onUseCache()
	}

	return nil
}

// GetEnv returns a map of the cumulative environment variables for all active virtual environments
func (v *VirtualEnvironment) GetEnv(inherit bool, projectDir string) map[string]string {
	var env map[string]string
	if v.getEnv == nil {
		logging.Error("setting up environment in un-activated project")
		env = make(map[string]string)
		env["PATH"] = os.Getenv("PATH")
	} else {
		var fail *failures.Failure
		env, fail = v.getEnv(inherit, projectDir)
		if fail != nil {
			logging.Error("could not set-up the runtime environment: %v", fail)
			return map[string]string{}
		}
	}

	if projectDir != "" {
		env[constants.ActivatedStateEnvVarName] = projectDir
		env[constants.ActivatedStateIDEnvVarName] = v.activationID
	}

	if inherit {
		return inheritEnv(env)
	}

	return env
}

// GetEnvSlice returns the same results as GetEnv, but formatted in a way that the process package can handle
func (v *VirtualEnvironment) GetEnvSlice(inherit bool) []string {
	envMap := v.GetEnv(inherit, filepath.Dir(projectfile.Get().Path()))
	var env []string
	for k, v := range envMap {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return env
}

// WorkingDirectory returns the working directory to use for the current environment
func (v *VirtualEnvironment) WorkingDirectory() string {
	wd, err := osutils.Getwd()
	if err != nil {
		// Shouldn't happen unless something is seriously wrong with your system
		panic(locale.T("panic_couldnt_detect_wd", map[string]interface{}{"Error": err.Error()}))
	}

	return wd
}

// ActivationID returns the unique identifier related to the activated instance.
func (v *VirtualEnvironment) ActivationID() string {
	return v.activationID
}
