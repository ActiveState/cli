package activate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/defact"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileevents"
	"github.com/ActiveState/cli/internal/hail"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type activationLoopFunc func(out output.Outputer, config defact.DefaultConfigurer, subs subshell.SubShell, targetPath string, setDefault bool, activator activateFunc) error

func activationLoop(out output.Outputer, config defact.DefaultConfigurer, subs subshell.SubShell, targetPath string, setDefault bool, activator activateFunc) error {
	// activate should be continually called while returning true
	// looping here provides a layer of scope to handle printing output
	var proj *project.Project
	for {
		var fail *failures.Failure
		proj, fail = project.FromPath(targetPath)
		if fail != nil {
			// The default failure returned by the project package is a big too vague, we want to give the user
			// something more actionable for the context they're in
			return failures.FailUserInput.New("err_project_from_path")
		}
		updater.PrintUpdateMessage(proj.Source().Path(), out)
		out.Notice(locale.T("info_activating_state", proj))

		if proj.CommitID() == "" {
			return errors.New(locale.Tr("err_project_no_commit", model.ProjectURL(proj.Owner(), proj.Name(), "")))
		}

		err := os.Chdir(targetPath)
		if err != nil {
			return err
		}

		if constants.BranchName != constants.StableBranch {
			out.Error(locale.Tr("unstable_version_warning", constants.BugTrackerURL))
		}

		keepGoing, err := activator(proj, out, config, subs, setDefault)
		if err != nil {
			return err
		}

		if !keepGoing {
			break
		}

		out.Notice(locale.T("info_reactivating", proj))
	}

	out.Notice(locale.T("info_deactivated", proj))

	return nil
}

type activateFunc func(proj *project.Project, out output.Outputer, config defact.DefaultConfigurer, subs subshell.SubShell, setDefault bool) (keepGoing bool, err error)

// activate will activate the venv and subshell. It is meant to be run in a loop
// with the return value indicating whether another iteration is warranted.
func activate(proj *project.Project, out output.Outputer, cfg defact.DefaultConfigurer, subs subshell.SubShell, setDefault bool) (bool, error) {
	projectfile.Reset()
	venv := virtualenvironment.Get()

	venv.OnDownloadArtifacts(func() { out.Notice(locale.T("downloading_artifacts")) })
	venv.OnInstallArtifacts(func() { out.Notice(locale.T("installing_artifacts")) })
	venv.OnUseCache(func() { out.Notice(locale.T("using_cached_env")) })

	activeProject := os.Getenv(constants.ActivatedStateEnvVarName)
	alreadyActivated := activeProject != ""

	// handle case, if we are already activated
	if alreadyActivated && !setDefault {
		return false, locale.NewError("err_already_active", "You cannot activate a new state when you are already in an activated state. You are in an activated state for project: {{.V0}}", proj.Name())
	}

	logging.Debug("Setting up virtual Environment")
	if strings.ToLower(os.Getenv(constants.DisableRuntime)) == "true" {
		logging.Debug("Skipping runtime activation")
	} else {
		fail := venv.Setup(true)
		if fail != nil {
			return false, locale.WrapError(fail, "error_could_not_activate_venv", "Could not activate project. If this is a private project ensure that you are authenticated.")
		}
	}

	if setDefault {
		logging.Debug("Setting up default activation for %s", proj.Name())
		err := defact.SetupDefaultActivation(cfg, out, venv)
		if err != nil {
			return false, locale.WrapError(err, "default_setup_err", "Failed to set up the default activation for {{.V0}}.", proj.Name())
		}
	}

	ignoreWindowsInterrupts()

	ve, err := venv.GetEnv(false, filepath.Dir(projectfile.Get().Path()))
	if err != nil {
		return false, locale.WrapError(err, "error_could_not_activate_venv", "Could not retrieve environment information.")
	}

	// If we're not using plain output then we should just dump the environment information
	if out.Type() != output.PlainFormatName {
		if out.Type() == output.EditorV0FormatName {
			fmt.Println("[activated-JSON]")
		}
		out.Print(ve)
		return false, nil
	}

	subs.SetEnv(ve)
	fail := subs.Activate(out)
	if fail != nil {
		return false, locale.WrapError(err, "error_could_not_activate_subshell", "Could not activate a new subshell.")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fname := path.Join(config.ConfigPath(), constants.UpdateHailFileName)

	hails, fail := hail.Open(ctx, fname)
	if fail != nil {
		return false, locale.WrapError(err, "error_unable_to_monitor_pulls", "Failed to setup pull monitoring")
	}

	fe, err := fileevents.New(proj)
	if err != nil {
		return false, locale.WrapError(err, "err_activate_fileevents", "Could not start file event watcher.")
	}
	defer fe.Close()

	return listenForReactivation(venv.ActivationID(), hails, subs)
}

func ignoreWindowsInterrupts() {
	if runtime.GOOS == "windows" {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT)
		go func() {
			for range c {
			}
		}()
	}
}

type subShell interface {
	Deactivate() *failures.Failure
	Failures() <-chan *failures.Failure
}

func listenForReactivation(id string, rcvs <-chan *hail.Received, subs subShell) (bool, error) {
	for {
		select {
		case rcvd, ok := <-rcvs:
			if !ok {
				return false, errs.New("hailing channel closed")
			}

			if rcvd.Fail != nil {
				logging.Error("error in hailing channel: %s", rcvd.Fail)
				continue
			}

			if !idsValid(id, rcvd.Data) {
				continue
			}

			if fail := subs.Deactivate(); fail != nil {
				return false, locale.WrapError(fail, "error_deactivating_subshell", "Failed to deactivate subshell properly")
			}

			// Wait for output completion after deactivating.
			// The nature of this issue is unclear at this time.
			time.Sleep(time.Second)

			return true, nil

		case fail, failed := <-subs.Failures():
			if !failed {
				return false, nil
			}

			if fail != nil {
				return false, locale.WrapError(fail, "error_in_active_subshell", "Failure encountered in active subshell")
			}

			return false, nil
		}
	}
}

func idsValid(currID string, rcvdID []byte) bool {
	return currID != "" && len(rcvdID) > 0 && currID == string(rcvdID)
}
