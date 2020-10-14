package virtualenvironment

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
)

var persisted *VirtualEnvironment

// FailAlreadyActive is a failure given when a project is already active
var FailAlreadyActive = failures.Type("virtualenvironment.fail.alreadyactive", failures.FailUser)

// VirtualEnvironment represents our virtual environment, it pulls together and virtualizes the runtime environment
type VirtualEnvironment struct {
	project      *project.Project
	activationID string
	onUseCache   func()
	runtime      *runtime.Runtime
}

// Init creates an instance of VirtualEnvironment{} with default settings
func Init() *VirtualEnvironment {
	return &VirtualEnvironment{
		activationID: uuid.New().String(),
	}
}

func New(runtime *runtime.Runtime) *VirtualEnvironment {
	return &VirtualEnvironment{
		activationID: uuid.New().String(),
		runtime:      runtime,
	}
}

// Activate the virtual environment
func (v *VirtualEnvironment) Activate() *failures.Failure {
	logging.Debug("Activating Virtual Environment")

	activeProject := os.Getenv(constants.ActivatedStateEnvVarName)
	if activeProject != "" {
		return FailAlreadyActive.New("err_already_active")
	}

	if strings.ToLower(os.Getenv(constants.DisableRuntime)) != "true" {
		if failure := v.Setup(true); failure != nil {
			return failure
		}
	}

	return nil
}

// OnUseCache will call the given function when the cached runtime is used
func (v *VirtualEnvironment) OnUseCache(f func()) { v.onUseCache = f }

// Setup sets up a runtime environment that is fully functional.
func (v *VirtualEnvironment) Setup(installIfNecessary bool) *failures.Failure {
	logging.Debug("Setting up virtual Environment")
	if strings.ToLower(os.Getenv(constants.DisableRuntime)) == "true" {
		return nil
	}
	if installIfNecessary {
		installer := runtime.NewInstaller(v.runtime)
		_, installed, fail := installer.Install()
		if fail != nil {
			return fail
		}

		if !installed && v.onUseCache != nil {
			v.onUseCache()
		}
	} else {
		_, fail := v.runtime.Env()
		if fail != nil {
			return fail
		}
	}

	return nil
}

// GetEnv returns a map of the cumulative environment variables for all active virtual environments
func (v *VirtualEnvironment) GetEnv(inherit bool, projectDir string) (map[string]string, error) {
	envMap := make(map[string]string)

	// Source runtime environment information
	if strings.ToLower(os.Getenv(constants.DisableRuntime)) != "true" {
		env, fail := v.runtime.Env()
		if fail != nil {
			return envMap, errs.Wrap(fail, "Could not initialize runtime env")
		}
		var err error
		envMap, err = env.GetEnv(inherit, projectDir)
		if err != nil {
			return envMap, err
		}
	}

	if projectDir != "" {
		envMap[constants.ActivatedStateEnvVarName] = projectDir
		envMap[constants.ActivatedStateIDEnvVarName] = v.activationID

		// Get project from explicitly defined configuration file
		pj, fail := project.Parse(filepath.Join(projectDir, constants.ConfigFileName))
		if fail != nil {
			return envMap, fail.ToError()
		}
		for _, constant := range pj.Constants() {
			var err error
			envMap[constant.Name()], err = constant.Value()
			if err != nil {
				return nil, locale.WrapError(err, "err_venv_constant_val", "Could not retrieve value for constant: `{{.V0}}`.", constant.Name())
			}
		}
	}

	if inherit {
		return inheritEnv(envMap), nil
	}

	return envMap, nil
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
