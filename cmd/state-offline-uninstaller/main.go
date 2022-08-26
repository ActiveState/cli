package main

import (
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
)

const (
	cmdUnInstall = "uninstall"
)

type Params struct {
	sourcePath string
	path       string
}

func newParams() *Params {
	return &Params{path: "/tmp"}
}

func main() {
	if os.Getenv("VERBOSE") == "true" {
		logging.CurrentHandler().SetVerbose(true)
	}

	runErr := run()
	if runErr != nil {
		errMsg := errs.Join(runErr, ": ").Error()
		if locale.IsInputError(runErr) {
			logging.Debug("state-offline-uninstaller errored out due to input: %s", errMsg)
		} else {
			logging.Critical("state-offline-uninstaller errored out: %s", errMsg)
		}

		fmt.Fprintln(os.Stderr, errMsg)
	}
}

func run() error {
	args := os.Args

	out, err := output.New("", &output.Config{
		OutWriter: os.Stdout,
		ErrWriter: os.Stderr,
	})
	if err != nil {
		return errs.Wrap(err, "Could not initialize outputer")
	}

	p := primer.New(nil, out, nil, nil, nil, nil, nil, nil, nil, nil)

	params := newParams()
        cmd := captain.NewCommand(
			cmdUnInstall,
			"Doing offline un-installation",
			"Do an offline un-installation",
			p, nil,
			[]*captain.Argument{
				{
					Name:        "path",
					Description: "Directory to uninstall <path>",
					Value:       &params.path,
					Required:    true,
				},
			},
			func(ccmd *captain.Command, args []string) error {
				logging.Debug("Running CmdUnInstall")
				return runOfflineUnInstall(out, params.path)
			},
	)

	return cmd.Execute(args[1:])
}
