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
	if project.Get() != nil && !subshell.IsActivated() {
		venv := virtualenvironment.Init()
		venv.OnDownloadArtifacts(func() {})
		venv.OnInstallArtifacts(func() {})

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
	if s.subshell.Binary() == "cmd" {
		lang = language.Batch
	}

	sf, fail := scriptfile.New(lang, fmt.Sprintf("state-shim-%s", args[0]), strings.Join(args, " "))
	if fail != nil {
		return locale.WrapError(fail.ToError(), "err_shim_create_scriptfile", "Could not generate script")
	}

	return s.subshell.Run(sf.Filename())
}
