package shim

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/scriptfile"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type Shim struct {
	subshell subshell.SubShell
}

type primeable interface {
	primer.Outputer
	primer.Subsheller
}

func New(prime primeable) *Shim {
	return &Shim{
		prime.Subshell(),
	}
}

func (s *Shim) Run(args ...string) error {
	project, fail := project.GetSafe()
	if fail != nil {
		// Do not fail if we can't find the projectfile
		logging.Debug("Project not found, error: %v", fail)
	}

	if project != nil && !subshell.IsActivated() {
		venv := virtualenvironment.Init()
		if fail := venv.Activate(); fail != nil {
			logging.Errorf("Unable to activate state: %s", fail.Error())
			return locale.WrapError(fail.ToError(), "err_shim_activate", "Could not activate environment for shim command")
		}

		env, err := venv.GetEnv(true, filepath.Dir(projectfile.Get().Path()))
		if err != nil {
			return err
		}
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

	sf, fail := scriptfile.New(lang, "state-shim", scriptArgs)
	if fail != nil {
		return locale.WrapError(fail.ToError(), "err_shim_create_scriptfile", "Could not generate script")
	}

	return s.subshell.Run(sf.Filename(), args[1:]...)
}
