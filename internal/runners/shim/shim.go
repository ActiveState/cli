package shim

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/path"
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

type Params struct {
	Script   string
	Language string
}

func New(prime primeable) *Shim {
	return &Shim{
		prime.Subshell(),
	}
}

func (s *Shim) Run(params Params, args ...string) error {
	var lang language.Language
	if params.Language != "" {
		lang = language.MakeByName(params.Language)
	} else {
		for _, l := range project.Get().Languages() {
			// Use first language found
			lang = language.MakeByName(l.Name())
			break
		}
	}
	if !lang.Recognized() {
		return locale.NewError("err_shim_language", "Unsupported language")
	}

	envPath := os.Getenv("PATH")
	if !subshell.IsActivated() {
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

		// get the "clean" path (only PATHS that are set by venv)
		env, err = venv.GetEnv(false, "")
		if err != nil {
			return err
		}
		envPath = env["PATH"]
	}

	if !path.ProvidesExecutable("", lang.Executable().Name(), envPath) {
		return locale.NewError("err_shim_exec", "Path does not contain: {{.V0}}", lang.Executable().Name())
	}

	scriptBlock, fail := fileutils.ReadFile(params.Script)
	if fail != nil {
		return locale.WrapError(fail.ToError(), "err_shim_read_script", "Could not read script file at: {{.V0}}", params.Script)
	}

	sf, fail := scriptfile.New(lang, filepath.Base(params.Script), string(scriptBlock))
	if fail != nil {
		return locale.WrapError(fail.ToError(), "err_shim_create_scriptfile", "Could not generate script")
	}

	return s.subshell.Run(sf.Filename(), args...)
}
