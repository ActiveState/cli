package virtualenvironment

import (
	"path/filepath"

	"github.com/google/uuid"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
)

var persisted *VirtualEnvironment

// VirtualEnvironment represents our virtual environment, it pulls together and virtualizes the runtime environment
type VirtualEnvironment struct {
	activationID string
	runtime      *runtime.Runtime
}

func New(runtime *runtime.Runtime) *VirtualEnvironment {
	return &VirtualEnvironment{
		activationID: uuid.New().String(),
		runtime:      runtime,
	}
}

// GetEnv returns a map of the cumulative environment variables for all active virtual environments
func (v *VirtualEnvironment) GetEnv(inherit bool, useExecutors bool, projectDir string) (map[string]string, error) {
	envMap := make(map[string]string)

	// Source runtime environment information
	if v.runtime != runtime.DisabledRuntime {
		var err error
		envMap, err = v.runtime.Env(inherit, useExecutors)
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
