package selfupdate

import (
	"os"

	"github.com/ActiveState/cli/pkg/projectfile"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/internal/updater"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/spf13/cobra"
)

// Command holds our main command definition
var Command = &commands.Command{
	Name:        "self-update",
	Description: "self-update",
	Run:         Execute,

	Flags: []*commands.Flag{
		&commands.Flag{
			Name:        "lock",
			Description: "flag_update_lock_description",
			Type:        commands.TypeBool,
			BoolVar:     &Flags.Lock,
		},
	},
}

// Flags hold the flag values passed through the command line.
var Flags struct {
	Lock bool
}

// Execute the current command
func Execute(cmd *cobra.Command, args []string) {
	if Flags.Lock {
		ExecuteLock(cmd, args)
		return
	}

	up := updater.Updater{
		CurrentVersion: constants.Version,
		APIURL:         constants.APIUpdateURL,
		Dir:            constants.UpdateStorageDir,
		CmdName:        constants.CommandName,
	}

	info, err := up.Info()
	if err != nil {
		failures.Handle(err, locale.T("err_no_update_info"))
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

	if isForwardCall() {
		// If this is a forward call (version locking) then we should just update the version in the activestate.yaml
		// The actual update will happen the next time the state tool is invoked in this project
		fail := lockVersion(info.Version)
		if fail != nil {
			failures.Handle(fail, locale.T("err_update_failed"))
			os.Exit(1)
		}
	} else {
		err = up.Run()
		if err != nil {
			failures.Handle(err, locale.T("err_update_failed"))
			os.Exit(1)
		}
	}

	print.Info(locale.T("update_complete"))
}

func ExecuteLock(cmd *cobra.Command, args []string) {
	print.Info(locale.Tr("locking_version", constants.Version))
	fail := lockVersion(constants.Version)
	if fail != nil {
		failures.Handle(fail, locale.T("err_lock_failed"))
		os.Exit(1)
	}
	print.Info(locale.Tr("version_locked", constants.Version))
}

func lockVersion(version string) *failures.Failure {
	pj := projectfile.Get()
	pj.Version = version
	return pj.Save()
}

func isForwardCall() bool {
	_, exists := os.LookupEnv(constants.ForwardedStateEnvVarName)
	return exists
}
