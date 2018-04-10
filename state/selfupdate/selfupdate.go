package selfupdate

import (
	"github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/ActiveState/ActiveState-CLI/internal/failures"
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/ActiveState/ActiveState-CLI/internal/updater"
	"github.com/ActiveState/ActiveState-CLI/pkg/cmdlets/commands"
	"github.com/spf13/cobra"
)

// Command holds our main command definition
var Command = &commands.Command{
	Name:        "self-update",
	Description: "self-update",
	Run:         Execute,
}

// Execute the current command
func Execute(cmd *cobra.Command, args []string) {
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

	if info == nil {
		print.Info(locale.T("no_update_available"))
		return
	}

	print.Info(locale.T("updating_to_version", map[string]interface{}{
		"fromVersion": constants.Version,
		"toVersion":   info.Version,
	}))

	err = up.Run()
	if err != nil {
		failures.Handle(err, locale.T("err_update_failed"))
		return
	}

	print.Info(locale.T("update_complete"))
}
