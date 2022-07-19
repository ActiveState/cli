package activate

import (
	"fmt"
	"os"
	"path/filepath"
	rt "runtime"

	anaConst "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileevents"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/process"
	"github.com/ActiveState/cli/internal/sighandler"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func (r *Activate) activateAndWait(proj *project.Project, venv *virtualenvironment.VirtualEnvironment) error {
	logging.Debug("Activating and waiting")

	err := os.Chdir(filepath.Dir(proj.Source().Path()))
	if err != nil {
		return err
	}

	ve, err := venv.GetEnv(false, true, filepath.Dir(projectfile.Get().Path()))
	if err != nil {
		return locale.WrapError(err, "error_could_not_activate_venv", "Could not retrieve environment information.")
	}

	if _, exists := os.LookupEnv(constants.DisableErrorTipsEnvVarName); exists {
		// If this exists, it came from the installer. It should not exist in an activated environment
		// otherwise.
		ve[constants.DisableErrorTipsEnvVarName] = "false"
	}

	// If we're not using plain output then we should just dump the environment information
	if r.out.Type() != output.PlainFormatName && r.out.Type() != output.SimpleFormatName {
		if r.out.Type() == output.EditorV0FormatName {
			fmt.Println("[activated-JSON]")
		}
		r.out.Print(ve)
		return nil
	}

	// ignore interrupts in State Tool on Windows
	if rt.GOOS == "windows" {
		bs := sighandler.NewBackgroundSignalHandler(func(_ os.Signal) {}, os.Interrupt)
		sighandler.Push(bs)
	}
	defer func() {
		if rt.GOOS == "windows" {
			sighandler.Pop()
		}
	}()

	r.subshell.SetEnv(ve)
	if err := r.subshell.Activate(proj, r.config, r.out); err != nil {
		return locale.WrapError(err, "error_could_not_activate_subshell", "Could not activate a new subshell.")
	}

	a, err := process.NewActivation(r.config, os.Getpid())
	if err != nil {
		return locale.WrapError(err, "error_could_not_mark_process", "Could not mark process as activated.")
	}
	defer a.Close()

	fe, err := fileevents.New(proj)
	if err != nil {
		return locale.WrapError(err, "err_activate_fileevents", "Could not start file event watcher.")
	}
	defer fe.Close()

	r.analytics.Event(anaConst.CatActivationFlow, "before-subshell")

	err = <-r.subshell.Errors()
	if err != nil {
		return locale.WrapError(err, "error_in_active_subshell", "Failure encountered in active subshell")
	}

	return nil
}
