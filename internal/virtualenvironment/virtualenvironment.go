package virtualenvironment

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
)

var persisted *VirtualEnvironment

// VirtualEnvironment represents our virtual environment, it pulls together and virtualizes the runtime environment
type VirtualEnvironment struct {
	activationID string
	onUseCache   func()
	runtime      *runtime.Runtime
}

func New(runtime *runtime.Runtime) *VirtualEnvironment {
	return &VirtualEnvironment{
		activationID: uuid.New().String(),
		runtime:      runtime,
	}
}

// Activate the virtual environment
func (v *VirtualEnvironment) Activate() error {
	logging.Debug("Activating Virtual Environment")

	if strings.ToLower(os.Getenv(constants.DisableRuntime)) != "true" {
		if err := v.Setup(true); err != nil {
			return err
		}
	}

	return nil
}

// OnUseCache will call the given function when the cached runtime is used
func (v *VirtualEnvironment) OnUseCache(f func()) { v.onUseCache = f }

// Setup sets up a runtime environment that is fully functional.
func (v *VirtualEnvironment) Setup(installIfNecessary bool) error {
	logging.Debug("Setting up virtual Environment")
	if strings.ToLower(os.Getenv(constants.DisableRuntime)) == "true" {
		return nil
	}
	if installIfNecessary {
		if !v.runtime.IsCachedRuntime() {
			installer := runtime.NewInstaller(v.runtime)
			_, _, err := installer.Install()
			if err != nil {
				return err
			}
		} else if v.onUseCache != nil {
			v.onUseCache()
		}
	} else {
		_, err := v.runtime.Env()
		if err != nil {
			return err
		}
	}

	return nil
}

// GetEnv returns a map of the cumulative environment variables for all active virtual environments
func (v *VirtualEnvironment) GetEnv(inherit bool, projectDir string) (map[string]string, error) {
	envMap := make(map[string]string)

	// Source runtime environment information
	if strings.ToLower(os.Getenv(constants.DisableRuntime)) != "true" {
		env, err := v.runtime.Env()
		if err != nil {
			return envMap, errs.Wrap(err, "Could not initialize runtime env")
		}
		envMap, err = env.GetEnv(inherit, projectDir)
		if err != nil {
			return envMap, err
		}
	}

	if projectDir != "" {
		envMap[constants.ActivatedStateEnvVarName] = projectDir
		envMap[constants.ActivatedStateIDEnvVarName] = v.activationID

		// Get project from explicitly defined configuration file
		pj, err := project.Parse(filepath.Join(projectDir, constants.ConfigFileName))
		if err != nil {
			return envMap, err
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
