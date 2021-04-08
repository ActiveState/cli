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

func main() {
	if len(os.Args) == 2 {
		timeout, err := strconv.Atoi(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot convert argument to number")
		}
		time.Sleep(time.Second * time.Duration(timeout))
		return
	}
	if len(os.Args) != 4 {
		fmt.Fprintf(os.Stderr, "Need to run with argument <from-dir> <installer> <logFile>")
		os.Exit(1)
	}

	fromDir := os.Args[1]
	installer := os.Args[2]
	logFile := os.Args[3]

	err := run(fromDir, installer, logFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", errs.Join(err, ":"))
		os.Exit(2)
	}
}

func run(fromDir, installer, logFile string) error {
	exe, err := osutils.Executable()
	if err != nil {
		return errs.Wrap(err, "Could not find executable path.")
	}
	toDir := filepath.Dir(exe)
	if err := exeutils.ExecuteAndForget(installer, fromDir, toDir, logFile); err != nil {
		return errs.Wrap(err, "Failed to run installer.")
	}

	// simulate some processing after starting the installer...
	time.Sleep(time.Second)
	return nil
}
