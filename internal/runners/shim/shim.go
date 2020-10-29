package shim

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type Shim struct {
	subshell subshell.SubShell
	proj     *project.Project
	out      output.Outputer
}

type primeable interface {
	primer.Outputer
	primer.Subsheller
	primer.Projecter
}

func New(prime primeable) *Shim {
	return &Shim{
		prime.Subshell(),
		prime.Project(),
		prime.Output(),
	}
}

func (s *Shim) Run(args ...string) error {
	if s.proj == nil {
		return locale.NewError("shim_no_project_found", "Could not find a project.  You need to be in a project directory or specify a global default project via `state activate --default`")
	}

	if len(args) == 0 {
		return nil
	}

	runtime, err := runtime.NewRuntime(s.proj.Source().Path(), s.proj.CommitUUID(), s.proj.Owner(), s.proj.Name(), runbits.NewRuntimeMessageHandler(s.out))
	if err != nil {
		return locale.WrapError(err, "err_shim_runtime_init", "Could not initialize runtime for shim command.")
	}
	venv := virtualenvironment.New(runtime)
	if fail := venv.Activate(); fail != nil {
		logging.Errorf("Unable to activate state: %s", fail.Error())
		return locale.WrapError(fail.ToError(), "err_shim_activate", "Could not activate environment for shim command")
	}

	env, err := venv.GetEnv(true, filepath.Dir(projectfile.Get().Path()))
	if err != nil {
		return err
	}
	logging.Debug("Trying to shim %s on PATH=%s", args[0], env["PATH"])
	// Ensure that we are not calling the shim recursively
	oldval, ok := env[constants.ShimEnvVarName]
	if ok && oldval == args[0] {
		return locale.NewError("err_shim_recursive_loop", "Could not resolve shimmed executable {{.V0}}", args[0])
	}
	env[constants.ShimEnvVarName] = args[0]

	s.subshell.SetEnv(env)

	lang := language.Bash
	scriptArgs := fmt.Sprintf(`%s "$@"`, args[0])
	if strings.Contains(s.subshell.Binary(), "cmd") {
		lang = language.Batch
		scriptArgs = fmt.Sprintf("@ECHO OFF\n%s %%*", args[0])
	}

	sf, fail := scriptfile.New(lang, "state-shim", scriptArgs)
	if fail != nil {
		return locale.WrapError(fail.ToError(), "err_shim_create_scriptfile", "Could not generate script")
	}

	return s.subshell.Run(sf.Filename(), args[1:]...)
}
