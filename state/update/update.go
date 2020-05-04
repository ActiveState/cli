package update

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
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
	},
}

// Flags hold the flag values passed through the command line.
var Flags struct {
	Lock bool
}

// Execute the current command
func Execute(cmd *cobra.Command, args []string) {
	var canBumpProject bool

	if !Flags.Lock { // targeting global
		if !isForwardCall() && projectfile.Get().Version == "" {
			updateGlobal() // NOTE: AC2: no-lock or locked project
			return
		}

		// TODO/NOTE: AC4 prompt to update locked version; skip the prompt if --force is set
		// TODO/NOTE: AC4: possibly address --quiet if --force handling is not sufficient

		// TODO: return if user resp is negative or continue out of this scope
		canBumpProject = true
	}

	updateProject(canBumpProject)
}

func updateProject(canBumpProject bool) {
	projectVersion := projectfile.Get().Version
	version := constants.Version

	// NOTE: AC1: with-lock just locks to the current version (Flags.Lock)
	// NOTE: AC3: no-lock updates locked version after prompt resp (!Flags.Lock && userAccepts)
	if canBumpProject && projectVersion != "" { // existing lock
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

	if fail := lockProjectVersion(constants.BranchName, version); fail != nil {
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

func lockProjectVersion(branch, version string) *failures.Failure {
	pj := projectfile.Get()
	pj.Branch = branch
	pj.Version = version
	return pj.Save()
}

func isForwardCall() bool {
	_, exists := os.LookupEnv(constants.ForwardedStateEnvVarName)
	return exists
}
