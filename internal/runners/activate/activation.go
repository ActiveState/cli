package activate

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/hail"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type activationLoopFunc func(targetPath string, activator activateFunc) error

func activationLoop(targetPath string, activator activateFunc) error {
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
		print.Info(locale.T("info_activating_state", proj))

		if proj.CommitID() == "" {
			return errors.New(locale.Tr("err_project_no_commit", model.ProjectURL(proj.Owner(), proj.Name(), "")))
		}

		err := os.Chdir(targetPath)
		if err != nil {
			return err
		}

		if constants.BranchName != constants.StableBranch {
			print.Stderr().Warning(locale.Tr("unstable_version_warning", constants.BugTrackerURL))
		}

		if !activator(proj.Owner(), proj.Name(), proj.Source().Path()) {
			break
		}

		print.Info(locale.T("info_reactivating", proj))
	}

	print.Bold(locale.T("info_deactivated", proj))

	return nil
}

type activateFunc func(owner, name, srcPath string) bool

// activate will activate the venv and subshell. It is meant to be run in a loop
// with the return value indicating whether another iteration is warranted.
func activate(owner, name, srcPath string) bool {
	projectfile.Reset()
	venv := virtualenvironment.Get()
	venv.OnDownloadArtifacts(func() { print.Line(locale.T("downloading_artifacts")) })
	venv.OnInstallArtifacts(func() { print.Line(locale.T("installing_artifacts")) })
	venv.OnUseCache(func() { print.Info(locale.T("using_cached_env")) })
	fail := venv.Activate()
	if fail != nil {
		failures.Handle(fail, locale.T("error_could_not_activate_venv"))
		return false
	}

	ignoreWindowsInterrupts()

	subs, fail := subshell.Get()
	if fail != nil {
		failures.Handle(fail, locale.T("error_could_not_activate_subshell"))
		return false
	}

	ve, err := venv.GetEnv(false, filepath.Dir(projectfile.Get().Path()))
	if err != nil {
		// wrapping error in failure, so Handle knows what to do with it...
		fail := failures.FailRuntime.Wrap(err)
		failures.Handle(fail, locale.T("error_could_not_activate_venv"))
		return false
	}

	subs.SetEnv(ve)
	fail = subs.Activate()
	if fail != nil {
		failures.Handle(fail, locale.T("error_could_not_activate_subshell"))
		return false
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	fname := path.Join(config.ConfigPath(), constants.UpdateHailFileName)

	hails, fail := hail.Open(ctx, fname)
	if fail != nil {
		failures.Handle(fail, locale.T("error_unable_to_monitor_pulls"))
		return false
	}

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

func listenForReactivation(id string, rcvs <-chan *hail.Received, subs subShell) bool {
	for {
		select {
		case rcvd, ok := <-rcvs:
			if !ok {
				logging.Error("hailing channel closed")
				return false
			}

			if rcvd.Fail != nil {
				logging.Error("error in hailing channel: %s", rcvd.Fail)
				continue
			}

			if !idsValid(id, rcvd.Data) {
				continue
			}

			if fail := subs.Deactivate(); fail != nil {
				failures.Handle(fail, locale.T("error_deactivating_subshell"))
				return false
			}

			// Wait for output completion after deactivating.
			// The nature of this issue is unclear at this time.
			time.Sleep(time.Second)

			return true

		case fail, ok := <-subs.Failures():
			if !ok {
				logging.Info("subshell failure channel closed")
				return false
			}

			if fail != nil {
				failures.Handle(fail, locale.T("error_in_active_subshell"))
			}

			return false
		}
	}
}

func idsValid(currID string, rcvdID []byte) bool {
	return currID != "" && len(rcvdID) > 0 && currID == string(rcvdID)
}
