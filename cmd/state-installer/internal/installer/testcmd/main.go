package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/exeutils"
	"github.com/ActiveState/cli/internal/osutils"
)

// testcmd can auto-update itself with a (reduced) installer build from ../testinst/main.go
// It can either sleep for a specified amount of time (to simulate being
// active), or call the installer which should replace this executable.

func main() {
	if len(os.Args) == 2 {
		timeout, err := strconv.Atoi(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot convert argument to number")
		}
		time.Sleep(time.Second * time.Duration(timeout))
		return
	}
	if len(os.Args) != 5 {
		fmt.Fprintf(os.Stderr, "Need to run with argument <from-dir> <installer> <logFile> <timeout>")
		os.Exit(1)
	}

	fromDir := os.Args[1]
	installer := os.Args[2]
	logFile := os.Args[3]
	timeout := os.Args[4]

	err := run(fromDir, installer, logFile, timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", errs.Join(err, ":"))
		os.Exit(2)
	}
}

func run(fromDir, installer, logFile, timeout string) error {
	exe, err := osutils.Executable()
	if err != nil {
		return errs.Wrap(err, "Could not find executable path.")
	}
	toDir := filepath.Dir(exe)
	proc, err := exeutils.ExecuteAndForget(installer, fromDir, toDir, logFile, timeout)
	if err != nil {
		return errs.Wrap(err, "Failed to run installer.")
	}
	if proc == nil {
		return errs.Wrap(err, "Failed to obtain process information.")
	}

	fmt.Printf("%d", proc.Pid)

	// simulate some shutdown sequence after starting the installer...
	time.Sleep(time.Second)
	return nil
}
