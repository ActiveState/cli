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
)

// Params tracks the info required for running Shim.
type Params struct {
	Path string
}

// NewParams constructs a Params and returns a pointer to it.
func NewParams() *Params {
	return &Params{}
}

type configurable interface {
	CachePath() string
}

// Shim manages the shim execution context.
type Shim struct {
	subshell subshell.SubShell
	proj     *project.Project
	out      output.Outputer
	cfg      configurable
}

type primeable interface {
	primer.Outputer
	primer.Subsheller
	primer.Projecter
	primer.Configurer
}

// New constructs a Shim and returns a pointer to it.
func New(prime primeable) *Shim {
	return &Shim{
		prime.Subshell(),
		prime.Project(),
		prime.Output(),
		prime.Config(),
	}
}

// Run executes the shim behavior.
func (s *Shim) Run(params *Params, args ...string) error {
	if params.Path != "" {
		var err error
		s.proj, err = project.FromPath(params.Path)
		if err != nil {
			return locale.WrapInputError(err, "shim_no_project_at_path", "Could not find project file at {{.V0}}", params.Path)
		}
	}
	if s.proj != nil {
		runtime, err := runtime.NewRuntime(s.proj.Source().Path(), s.cfg.CachePath(), s.proj.CommitUUID(), s.proj.Owner(), s.proj.Name(), runbits.NewRuntimeMessageHandler(s.out))
		if err != nil {
			return locale.WrapError(err, "err_shim_runtime_init", "Could not initialize runtime for shim command.")
		}
		venv := virtualenvironment.New(runtime)
		if err := venv.Activate(); err != nil {
			logging.Errorf("Unable to activate state: %s", err.Error())
			return locale.WrapError(err, "err_shim_activate", "Could not activate environment for shim command")
		}

		env, err := venv.GetEnv(true, filepath.Dir(s.proj.Source().Path()))
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
	}

	if len(args) == 0 {
		return nil
	}

	lang := language.Bash
	scriptArgs := fmt.Sprintf(`%s "$@"`, args[0])
	if strings.Contains(s.subshell.Binary(), "cmd") {
		lang = language.Batch
		scriptArgs = fmt.Sprintf("@ECHO OFF\n%s %%*", args[0])
	}

	sf, err := scriptfile.New(lang, "state-shim", scriptArgs)
	if err != nil {
		return locale.WrapError(err, "err_shim_create_scriptfile", "Could not generate script")
	}

	return s.subshell.Run(sf.Filename(), args[1:]...)
}
