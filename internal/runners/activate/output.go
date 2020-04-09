package activate

import (
	"encoding/json"
	"os"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/runtime"
)

func activateOutput(targetPath string, output commands.Output) error {
	err := os.Chdir(targetPath)
	if err != nil {
		return err
	}

	jsonString, err := envOutput(false, targetPath)
	if err != nil {
		if output == commands.EditorV0 {
			return updateOutputError(err)
		}
		return err
	}

	print.Line("[activated-JSON]")
	print.Line(jsonString)
	return nil
}

func envOutput(inherit bool, targetPath string) (string, error) {
	venv := virtualenvironment.Get()
	fail := venv.Activate()
	if fail != nil {
		return "", fail
	}

	env := venv.GetEnv(inherit, targetPath)
	envJSON, err := json.Marshal(env)
	if err != nil {
		return "", err
	}

	return string(envJSON), nil
}

func updateOutputError(err error) error {
	fail, ok := err.(*failures.Failure)
	if !ok {
		return err
	}

	switch fail.Type {
	case runtime.FailBuildFailed:
		return runtime.FailBuildFailed.New(locale.T("err_activate_output_build_failed"))
	case runtime.FailBuildInProgress:
		return runtime.FailBuildInProgress.New(locale.T("err_activate_output_build_in_progress"))
	default:
		return fail
	}
}
