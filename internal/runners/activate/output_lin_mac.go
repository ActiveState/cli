// +build !windows

package activate

import (
	"encoding/json"

	"github.com/ActiveState/cli/internal/virtualenvironment"
)

func envOutput() (string, error) {
	venv := virtualenvironment.Get()
	fail := venv.Activate()
	if fail != nil {
		return "", fail
	}

	env := venv.GetEnv(true)
	envJSON, err := json.Marshal(env)
	if err != nil {
		return "", err
	}

	return string(envJSON), nil
}
