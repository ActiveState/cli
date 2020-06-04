package virtualenvironment

import (
	"fmt"
	"os"
	"path/filepath"
	rt "runtime"
	"strings"

	"github.com/google/uuid"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
)

var persisted *VirtualEnvironment

// OS is used by tests to spoof a different value
var OS = rt.GOOS

type getEnvFunc func(inherit bool, projectDir string) (map[string]string, error)

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
func (v *VirtualEnvironment) Activate() error {
	logging.Debug("Activating Virtual Environment")

	activeProject := os.Getenv(constants.ActivatedStateEnvVarName)
	if activeProject != "" {
		return locale.NewError("err_already_active", "You cannot activate a new state when you are already in an activated state. You are in an activated state for project: {{.V0}}", v.project.Owner()+"/"+v.project.Name())
	}

	fmt.Println("DISABLE_RUNTIME set? ", strings.ToLower(os.Getenv(constants.DisableRuntime)))
	if strings.ToLower(os.Getenv(constants.DisableRuntime)) != "true" {
		if err := v.activateRuntime(); err != nil {
			return err
		}
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
func (v *VirtualEnvironment) activateRuntime() error {
	pj := project.Get()
	installer, err := runtime.NewInstaller(pj.CommitUUID(), pj.Owner(), pj.Name())
	if err != nil {
		return err
	}

	installer.OnDownload(v.onDownloadArtifacts)

	rt, installed, err := installer.Install()
	fmt.Printf("in activateRuntime: %v %v %v\n", rt, installed, err)
	if err != nil {
		return err
	}

	v.getEnv = rt.GetEnv
	if !installed && v.onUseCache != nil {
		v.onUseCache()
	}

	return nil
}

// GetEnv returns a map of the cumulative environment variables for all active virtual environments
func (v *VirtualEnvironment) GetEnv(inherit bool, projectDir string) (map[string]string, error) {
	env := make(map[string]string)
	if v.getEnv == nil {
		// if runtime is not explicitly disabled, this is an error
		if os.Getenv(constants.DisableRuntime) != "true" {
			return nil, locale.NewError(
				"err_get_env_unactivated", "Trying to set up an environment in an un-activated environment.  This should not happen.  Please report this issue in our forum: %s",
				constants.ForumsURL,
			)
		}
		env["PATH"] = os.Getenv("PATH")
	} else {
		var err error
		env, err = v.getEnv(inherit, projectDir)
		if err != nil {
			return env, err
		}
	}

	if projectDir != "" {
		env[constants.ActivatedStateEnvVarName] = projectDir
		env[constants.ActivatedStateIDEnvVarName] = v.activationID

		// Get project from explicitly defined configuration file
		pj, fail := project.Parse(filepath.Join(projectDir, constants.ConfigFileName))
		if fail != nil {
			return env, fail.ToError()
		}
		for _, constant := range pj.Constants() {
			env[constant.Name()] = constant.Value()
		}
	}

	if inherit {
		return inheritEnv(env), nil
	}

	return env, nil
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
