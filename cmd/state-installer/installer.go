package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, errs.Join(err, ": ").Error())
		os.Exit(1)
	}
}

func run() error {
	exe, err := osutils.Executable()
	if err != nil {
		return errs.Wrap(err, "Could not detect executable path")
	}

	target :=
	if len(os.Args) <= 1 {

	}

	for _, file := range fileutils.ListDir(filepath.Dir(exe), false) {

	}
}
