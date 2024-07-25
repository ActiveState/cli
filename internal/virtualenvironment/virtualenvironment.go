package virtualenvironment

import (
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/ActiveState/cli/pkg/runtime"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/pkg/project"
)

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
func (v *VirtualEnvironment) GetEnv(inherit bool, useExecutors bool, projectDir, namespace string) (map[string]string, error) {
	envMap := make(map[string]string)

	// Source runtime environment information

	env := v.runtime.Env(inherit)
	if useExecutors {
		envMap = env.VariablesWithExecutors
	} else {
		envMap = env.Variables
	}

	if projectDir != "" {
		envMap[constants.ActivatedStateEnvVarName] = projectDir
		envMap[constants.ActivatedStateIDEnvVarName] = v.activationID
		envMap[constants.ActivatedStateNamespaceEnvVarName] = namespace

		// Get project from explicitly defined configuration file
		configFile := filepath.Join(projectDir, constants.ConfigFileName)
		pj, err := project.Parse(configFile)
		if err != nil {
			return envMap, locale.WrapError(err, "err_parse_project", "", configFile)
		}
		for _, constant := range pj.Constants() {
			v, err := constant.Value()
			envMap[constant.Name()] = strings.Replace(v, "\n", `\n`, -1)
			if err != nil {
				return nil, locale.WrapError(err, "err_venv_constant_val", "Could not retrieve value for constant: `{{.V0}}`.", constant.Name())
			}
		}
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
