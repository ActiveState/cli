package update

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type Params struct {
	Lock  bool
	Force bool
}

type Update struct {
	project *project.Project
	out     output.Outputer
}

func New(pj *project.Project, out output.Outputer) *Update {
	return &Update{
		pj,
		out,
	}
}

func (u *Update) Run(params *Params) error {
	return run(params.Lock, isLocked(), params.Force, u.runLock, u.runUpdateLock, u.runUpdateGlobal, confirmUpdateLock)
}

func run(lock, isLocked, force bool, runLock, runUpdateLock, runUpdateGlobal, confirmLock func() error) error {
	if lock {
		return runLock()
	}
	if !lock && isLocked {
		if !force {
			if err := confirmLock(); err != nil {
				return locale.WrapError(err, "err_update_lock_confirm", "Could not confirm whether to update.")
			}
		}
		return runUpdateLock()
	}
	return runUpdateGlobal()
}

func (u *Update) runLock() error {
	u.out.Notice(locale.Tl("locking_version", "Locking State Tool to the current version."))

	if u.project.Version() != "" {
		u.out.Print(locale.Tl("lock_project_uptodate", "Your project is already locked, did you mean to run 'state update' (without the --lock flag)?"))
		return nil
	}

	err := u.setUpdateInYAML(constants.Version, constants.BranchName)
	if err != nil {
		return locale.WrapError(err, "err_update_projectfile", "Could not update projectfile")
	}

	u.out.Print(locale.Tl("version_locked", "Version locked at {{.V0}}", constants.Version))
	return nil
}

func (u *Update) runUpdateLock() error {
	u.out.Notice(locale.Tl("updating_lock_version", "Locking State Tool to latest version available for your project."))

	info, err := updater.New(u.project.Version()).Info()
	if err != nil {
		return locale.WrapError(err, "err_update_updater", "Could not retrieve update information.")
	}

	if info == nil {
		u.out.Print(locale.Tl("update_project_uptodate", "Your project is already using the latest State Tool version available."))
		return nil
	}

	err = u.replaceUpdateInYAML(info.Version, "master")
	if err != nil {
		return locale.WrapError(err, "err_update_projectfile", "Could not replace update in projectfile")
	}

	u.out.Print(locale.Tl("version_lock_updated", "Locked version updated to {{.V0}}", constants.Version))
	return nil
}

func (u *Update) setUpdateInYAML(version, branch string) error {
	data, err := ioutil.ReadFile(u.project.Source().Path())
	if err != nil {
		return locale.WrapError(err, "err_read_projectfile", "Failed to read the activestate.yaml at: %s", u.project.Source().Path())
	}

	projectRegex := regexp.MustCompile(`(?m:project:.+\n)`)
	index := projectRegex.FindIndex(data)
	if len(index) != 2 {
		// The second index returned represents the newline character at the end of the 'project:' entry in the activestate.yaml
		// which is where we want to insert the lock information
		return locale.NewError("err_find_project_entry", "Could not find valid project entry in projectfile")
	}

	return fileutils.InsertIntoFile(u.project.Source().Path(), index[1], []byte(fmt.Sprintf("branch: %s\nversion: %s\n", branch, version)))
}

func (u *Update) replaceUpdateInYAML(version, branch string) error {
	data, err := ioutil.ReadFile(u.project.Source().Path())
	if err != nil {
		return locale.WrapError(err, "err_read_projectfile", "Failed to read the activestate.yaml at: %s", u.project.Source().Path())
	}

	branchRegex := regexp.MustCompile(`(?m:(branch:\s*)(\w+))`)
	branchUpdate := []byte(fmt.Sprintf("${1}%s", branch))
	out := branchRegex.ReplaceAll(data, branchUpdate)

	versionRegex := regexp.MustCompile(`(?m:(version:\s*)(\d+.\d+.\d+-[A-Za-z0-9]+))`)
	versionUpdate := []byte(fmt.Sprintf("${1}%s", version))

	replaced := versionRegex.ReplaceAll(out, versionUpdate)

	return ioutil.WriteFile(u.project.Source().Path(), replaced, 0644)
}

func (u *Update) runUpdateGlobal() error {
	u.out.Notice(locale.Tl("updating_version", "Updating State Tool to latest version available."))

	up := updater.New(constants.Version)
	info, err := up.Info()
	if err != nil {
		return locale.WrapError(err, "err_update_updater", "Could not retrieve update information.")
	}

	if info == nil {
		u.out.Print(locale.Tl("update_uptodate", "You are already using the latest State Tool version available."))
		return nil
	}

	if err = up.Run(u.out); err != nil {
		if os.IsPermission(errs.InnerError(err)) {
			return locale.WrapError(err, "err_update_failed_due_to_permissions", "Update failed due to permission error.  You may have to re-run the command as a privileged user.")
		}
		return locale.WrapError(err, "err_update_failed", "Update failed, please try again later or try reinstalling the State Tool.")
	}

	u.out.Print(locale.Tl("version_updated", "Version updated to {{.V0}}", info.Version))
	return nil
}

func confirmUpdateLock() error {
	msg := locale.T("confirm_update_locked_version_prompt")

	prom := prompt.New()
	confirmed, fail := prom.Confirm(msg, false)
	if fail != nil {
		return fail.ToError()
	}

	if !confirmed {
		return locale.NewInputError("err_update_lock_noconfirm", "Cancelling by your request.")
	}

	return nil
}

func isLocked() bool {
	pj, fail := projectfile.GetSafe()
	return fail == nil && pj.Version != ""
}
