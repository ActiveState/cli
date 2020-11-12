package activate

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	rt "runtime"
	"syscall"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/fileevents"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/process"
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

	ve, err := venv.GetEnv(false, filepath.Dir(projectfile.Get().Path()))
	if err != nil {
		return locale.WrapError(err, "error_could_not_activate_venv", "Could not retrieve environment information.")
	}

	// If we're not using plain output then we should just dump the environment information
	if r.out.Type() != output.PlainFormatName {
		if r.out.Type() == output.EditorV0FormatName {
			fmt.Println("[activated-JSON]")
		}
		r.out.Print(ve)
		return nil
	}

	ignoreWindowsInterrupts()

	err = r.config.WriteConfig()
	if err != nil {
		return locale.WrapError(err, "err_write_config", "Could not write to configuration file")
	}

	r.subshell.SetEnv(ve)
	if fail := r.subshell.Activate(r.out); fail != nil {
		return locale.WrapError(fail, "error_could_not_activate_subshell", "Could not activate a new subshell.")
	}

	a, err := process.NewActivation(os.Getpid())
	if err != nil {
		return locale.WrapError(err, "error_could_not_mark_process", "Could not mark process as activated.")
	}
	defer a.Close()

	fe, err := fileevents.New(proj)
	if err != nil {
		return locale.WrapError(err, "err_activate_fileevents", "Could not start file event watcher.")
	}
	defer fe.Close()

	analytics.Event(analytics.CatActivationFlow, "before-subshell")

	err = <-r.subshell.Errors()
	if err != nil {
		return locale.WrapError(err, "error_in_active_subshell", "Failure encountered in active subshell")
	}

	return nil
}

func ignoreWindowsInterrupts() {
	if rt.GOOS == "windows" {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT)
		go func() {
			for range c {
			}
		}()
	}
}
