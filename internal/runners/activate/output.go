package activate

import (
	"encoding/json"
	"os"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/runtime"
)

func activateOutput(targetPath string, outfmt output.Format) error {
	err := os.Chdir(targetPath)
	if err != nil {
		return err
	}

	jsonString, err := envOutput(false)
	if err != nil {
		if outfmt == output.EditorV0 {
			return updateOutputError(err)
		}
		return err
	}

	print.Line("[activated-JSON]")
	print.Line(jsonString)
	return nil
}

func envOutput(inherit bool) (string, error) {
	venv := virtualenvironment.Get()
	fail := venv.Activate()
	if fail != nil {
		return "", fail
	}

	env := venv.GetEnv(inherit)
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
