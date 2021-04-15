package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/ActiveState/cli/internal/installation"
	"github.com/ActiveState/cli/internal/logging"
)

// testinst is a reduced version of the state-installer. It only installs files
// from a given directory but does not manage state-svc and state-tray apps.
// This programme is used in the TestAutoUpdate() test

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "Need to run with argument <from-dir> <to-dir> <timeout>")
		os.Exit(1)
	}
	fromDir := os.Args[1]
	toDir := os.Args[2]
	timeout, err := strconv.Atoi(os.Args[3])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse timeout %s: %v", os.Args[3], err)
		os.Exit(1)
	}
	logging.CurrentHandler().SetVerbose(true)

	// pausing before installation (to give time to stop running executables)
	time.Sleep(time.Duration(timeout) * time.Second)

	err = run(fromDir, toDir)
	if err != nil {
		logging.Debug("Installation failed: %v", err)
		os.Exit(1)
	}
	logging.Debug("Installation from %s -> %s was successful.", fromDir, toDir)

}

func run(fromDir, toDir string) error {
	logging.Debug("Installing %s -> %s", fromDir, toDir)
	inst, err := installation.New(fromDir, toDir)
	if err != nil {
		return err
	}
	defer inst.Close()

	if err := inst.Install(); err != nil {
		if restErr := inst.RestoreBackup(); restErr != nil {
			logging.Error("Restoring the backup failed: %v", restErr)
		}
		return err
	}
	return nil
}
