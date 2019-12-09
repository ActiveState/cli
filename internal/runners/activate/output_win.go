// +build windows

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

	env := virtualenvironment.Get().GetEnv(true)
	envJSON, err := json.Marshal(env)
	if err != nil {
		return "", err
	}

	return string(envJSON), nil
}
