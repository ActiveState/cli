package update

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// Command holds our main command definition
var Command = &commands.Command{
	Name:        "update",
	Description: "update_description",
	Run:         Execute,

	Flags: []*commands.Flag{
		&commands.Flag{
			Name:        "lock",
			Description: "flag_update_lock_description",
			Type:        commands.TypeBool,
			BoolVar:     &Flags.Lock,
		},
		&commands.Flag{
			Name:        "force",
			Description: "flag_update_force_description",
			Type:        commands.TypeBool,
			BoolVar:     &Flags.Force,
		},
	},
}

// Flags hold the flag values passed through the command line.
var Flags struct {
	Lock  bool
	Force bool
}

// Execute the current command
func Execute(cmd *cobra.Command, args []string) {
	var updateFirst bool

	if !Flags.Lock {
		if !isForwardedOrLocked() {
			updateGlobal()
			return
		}

		if fail := confirmUpdateLocked(Flags.Force); fail != nil {
			failures.Handle(fail, locale.T("err_lock_failed"))
			return
		}

		updateFirst = true
	}

	lockProject(updateFirst)
}

func confirmUpdateLocked(force bool) *failures.Failure {
	if force {
		return nil
	}

	msg := locale.T("confirm_update_locked_version_prompt")

	prom := prompt.New()
	confirmed, fail := prom.Confirm(msg, false)
	if fail != nil {
		return fail
	}

	if !confirmed {
		return failures.FailUserInput.New("err_action_was_not_confirmed")
	}

	return nil
}

func isForwardedOrLocked() bool {
	if isForwardCall() {
		return true
	}

	pj, fail := projectfile.GetSafe()

	return fail == nil && pj.Version != ""
}

func lockProject(updateFirst bool) {
	pj, fail := projectfile.GetSafe()
	if fail != nil {
		failures.Handle(fail, locale.T("err_lock_failed"))
		return
	}

	projectVersion := pj.Version
	version := constants.Version

	if updateFirst && projectVersion != "" { // existing lock
		info, err := newUpdater(projectVersion).Info()
		if err != nil {
			failures.Handle(err, locale.T("err_no_update_info"))
			return
		}

		logging.Debug("Update info: %v", info)
		logging.Debug("Current version: %s", projectVersion)

		if info == nil {
			print.Info(locale.T("no_update_available"))
			return
		}

		version = info.Version

		print.Info(locale.T("updating_to_version", map[string]interface{}{
			"fromVersion": projectVersion,
			"toVersion":   version,
		}))
	} else {
		print.Info(locale.Tr("locking_version", version))
	}

	if fail := setProjectVersion(constants.BranchName, version); fail != nil {
		failures.Handle(fail, locale.T("err_lock_failed"))
		return
	}

	print.Info(locale.Tr("version_locked", version))
}

func updateGlobal() {
	up := newUpdater(constants.Version)
	info, err := up.Info()
	if err != nil {
		failures.Handle(err, locale.T("err_no_update_info"))
		return
	}

	logging.Debug("Update info: %v", info)
	logging.Debug("Current version: %s", constants.Version)

	if info == nil {
		print.Info(locale.T("no_update_available"))
		return
	}

	print.Info(locale.T("updating_to_version", map[string]interface{}{
		"fromVersion": constants.Version,
		"toVersion":   info.Version,
	}))

	if err = up.Run(); err != nil {
		failures.Handle(err, locale.T("err_update_failed"))
		return
	}

	print.Info(locale.T("update_complete"))
}

// leave desiredVersion empty for latest
func newUpdater(currentVersion string) *updater.Updater {
	return &updater.Updater{
		CurrentVersion: currentVersion,
		APIURL:         constants.APIUpdateURL,
		Dir:            constants.UpdateStorageDir,
		CmdName:        constants.CommandName,
	}
}

func setProjectVersion(branch, version string) *failures.Failure {
	pj, fail := projectfile.GetSafe()
	if fail != nil {
		return fail
	}

	pj.Branch = branch
	pj.Version = version

	return pj.Save()
}

func isForwardCall() bool {
	_, exists := os.LookupEnv(constants.ForwardedStateEnvVarName)
	return exists
}
