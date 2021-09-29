package exec

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/globaldefault"
	"github.com/ActiveState/cli/internal/installation/storage"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup"
	"github.com/ActiveState/cli/pkg/project"
)

type Exec struct {
	subshell  subshell.SubShell
	proj      *project.Project
	auth      *authentication.Auth
	out       output.Outputer
	cfg       *config.Instance
	analytics analytics.AnalyticsDispatcher
}

type primeable interface {
	primer.Auther
	primer.Outputer
	primer.Subsheller
	primer.Projecter
	primer.Configurer
	primer.Analyticer
}

type Params struct {
	Path string
}

func New(prime primeable) *Exec {
	return &Exec{
		prime.Subshell(),
		prime.Project(),
		prime.Auth(),
		prime.Output(),
		prime.Config(),
		prime.Analytics(),
	}
}

func NewParams() *Params {
	return &Params{}
}

func (s *Exec) Run(params *Params, args ...string) error {
	var projectDir string
	var rtTarget setup.Targeter

	// Detect target and project dir
	// If the path passed resolves to a runtime dir (ie. has a runtime marker) then the project is not used
	if params.Path != "" && runtime.IsRuntimeDir(params.Path) {
		rtTarget = runtime.NewCustomTarget("", "", "", params.Path)
	} else {
		proj := s.proj
		if params.Path != "" {
			var err error
			proj, err = project.FromPath(params.Path)
			if err != nil {
				return locale.WrapInputError(err, "exec_no_project_at_path", "Could not find project file at {{.V0}}", params.Path)
			}
		}
		if s.proj == nil {
			return locale.NewError("exec_no_project_found", "Could not find a project.  You need to be in a project directory or specify a global default project via `state activate --default`")
		}
		projectDir = filepath.Dir(proj.Source().Path())
		rtTarget = runtime.NewProjectTarget(proj, storage.CachePath(), nil)
	}

	if len(args) == 0 {
		return nil
	}

	rt, err := runtime.New(rtTarget, s.analytics)
	if err != nil {
		if !runtime.IsNeedsUpdateError(err) {
			return locale.WrapError(err, "err_activate_runtime", "Could not initialize a runtime for this project.")
		}
		eh, err := runbits.DefaultRuntimeEventHandler(s.out)
		if err != nil {
			return locale.WrapError(err, "err_initialize_runtime_event_handler")
		}
		if err := rt.Update(s.auth, eh); err != nil {
			return locale.WrapError(err, "err_update_runtime", "Could not update runtime installation.")
		}
	}
	venv := virtualenvironment.New(rt)

	env, err := venv.GetEnv(true, false, projectDir)
	if err != nil {
		return locale.WrapError(err, "err_exec_env", "Could not retrieve environment information for your runtime")
	}
	logging.Debug("Trying to exec %s on PATH=%s", args[0], env["PATH"])
	// Ensure that we would not call the executor recursively: The path for the executable should be different from the default bin dir
	p := exeutils.FindExecutableOnOSPath(filepath.Base(args[0]))
	binDir := filepath.Clean(globaldefault.BinDir(s.cfg))
	if p == binDir {
		return logging.Criticalf("Detected recursive loop while calling %s", args[0])
	}

	s.subshell.SetEnv(env)

	lang := language.Bash
	scriptArgs := fmt.Sprintf(`%s "$@"`, args[0])
	if strings.Contains(s.subshell.Binary(), "cmd") {
		lang = language.Batch
		scriptArgs = fmt.Sprintf("@ECHO OFF\n%s %%*", args[0])
	}

	sf, err := scriptfile.New(lang, "state-exec", scriptArgs)
	if err != nil {
		return locale.WrapError(err, "err_exec_create_scriptfile", "Could not generate script")
	}

	return s.subshell.Run(sf.Filename(), args[1:]...)
}
