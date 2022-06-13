package main

import (
	"fmt"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/subshell/sscommon"
)

const testDir = `C:\test`

func main() {
	err := run()
	if err != nil {
		fmt.Println("Could not run:", errs.JoinMessage(err))
		return
	}
}

func run() error {
	cfg, err := config.New()
	if err != nil {
		return errs.Wrap(err, "Could not get config")
	}

	isAdmin, err := osutils.IsAdmin()
	if err != nil {
		return errs.Wrap(err, "Could not check if admin")
	}

	shell := subshell.New(cfg)
	err = shell.CleanUserEnv(cfg, sscommon.InstallID, !isAdmin)
	if err != nil {
		return errs.Wrap(err, "Could not update PATH")
	}

	return nil

}
