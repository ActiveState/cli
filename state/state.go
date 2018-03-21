package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/ActiveState/ActiveState-CLI/internal/config" // MUST be first!
	"github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/ActiveState/ActiveState-CLI/internal/locale"
	"github.com/ActiveState/ActiveState-CLI/internal/print"
	"github.com/ActiveState/ActiveState-CLI/internal/updater"
	"github.com/ActiveState/ActiveState-CLI/pkg/cmdlets/commands"

	// commands
	"github.com/ActiveState/ActiveState-CLI/state/activate"
	"github.com/ActiveState/ActiveState-CLI/state/hook"
	"github.com/ActiveState/ActiveState-CLI/state/selfupdate"

	"github.com/ActiveState/ActiveState-CLI/internal/logging"
	"github.com/spf13/cobra"
)

var exit = os.Exit

// T links to locale.T
var T = locale.T

// Flags hold the flag values passed through the command line
var Flags struct {
	Locale  string
	Version bool
}

// Command holds our main command definition
var Command = &commands.Command{
	Name:        "state",
	Description: "state_description",
	Run:         Execute,

	Flags: []*commands.Flag{
		&commands.Flag{
			Name:        "locale",
			Shorthand:   "l",
			Description: "flag_state_locale_description",
			Type:        commands.TypeString,
			Persist:     true,
			StringVar:   &Flags.Locale,
		},
		&commands.Flag{
			Name:        "version",
			Description: "flag_state_version_description",
			Type:        commands.TypeBool,
			BoolVar:     &Flags.Version,
		},
	},

	UsageTemplate: "usage_tpl",
}

func init() {
	logging.Debug("init")

	Command.Append(activate.Command)
	Command.Append(hook.Command)
	Command.Append(selfupdate.Command)
}

func main() {
	logging.Debug("main")

	if checkForAndApplyUpdates() {
		relaunch() // will not return
	}

	// This actually runs the command
	err := Command.Execute()

	if err != nil {
		fmt.Println(err)
		exit(1)
		return
	}

	// Write our config to file
	config.Save()
}

// Execute the `state` command
func Execute(cmd *cobra.Command, args []string) {
	logging.Debug("Execute")

	if Flags.Version {
		print.Info(locale.T("version_info", map[string]interface{}{"Version": constants.Version}))
		return
	}

	cmd.Usage()
}

// Checks for updates once per day and, if one was found, applies it and returns
// `true`. Otherwise, returns `false`.
func checkForAndApplyUpdates() bool {
	// Determine whether or not an update check has been performed today.
	updateCheckMarker := filepath.Join(os.TempDir(), "activestate-cli-update-check")
	marker, err := os.Stat(updateCheckMarker)
	if err != nil {
		// Marker does not exist. Create it.
		err = ioutil.WriteFile(updateCheckMarker, []byte(""), 0666)
		if err != nil {
			logging.Debug("Unable to automatically check for updates: %s", err)
			return false
		}
	} else {
		// Check to see if it has been 24 hours since the last update check. If not,
		// skip another check.
		nextCheckTime := marker.ModTime().Add(24 * time.Hour)
		if time.Now().Before(nextCheckTime) {
			logging.Debug("Not checking for updates until %s", nextCheckTime)
			return false
		}
	}
	// Will check for updates. Touch the update check marker so the next check
	// will not happen for another day.
	err = os.Chtimes(updateCheckMarker, time.Now(), time.Now())
	if err != nil {
		logging.Debug("Unable to automatically check for updates: %s", err)
		return false
	}

	// Check for an update.
	print.Info(locale.T("checking_for_updates"))
	update := updater.Updater{
		CurrentVersion: constants.Version,
		APIURL:         constants.APIUpdateURL,
		Dir:            constants.UpdateStorageDir,
		CmdName:        constants.CommandName,
	}
	info, err := update.Info()
	if err != nil {
		logging.Debug("Unable to automatically check for updates: %s", err)
		return false
	} else if info == nil {
		print.Info(locale.T("no_update_available"))
		return false
	}
	print.Info(locale.T("updating_to_version", map[string]interface{}{
		"fromVersion": constants.Version,
		"toVersion":   info.Version,
	}))

	// Self-update.
	err = update.Run()
	if err != nil {
		logging.Debug("Unable to automatically check for updates: %s", err)
		return false
	}
	print.Info(locale.T("update_complete"))

	return true
}

// When an update was found and applied, re-launch the update with the current
// arguments and wait for return before exitting.
// This function will never return to its caller.
func relaunch() {
	var wg sync.WaitGroup
	wg.Add(1)
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Start()
	var err error
	go func() {
		err = cmd.Wait()
		if err != nil {
			panic(err.Error())
		}
		wg.Done()
	}()
	wg.Wait()
	os.Exit(0)
}
