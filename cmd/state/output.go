package main

import (
	"errors"
	"os"

	"github.com/jessevdk/go-flags"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/terminal"

	survey "gopkg.in/AlecAivazis/survey.v1/core"
)

type outputFlags struct {
	// These should be kept in sync with cmd/state/internal/cmdtree (output flag)
	Output         string `short:"o" long:"output"`
	Mono           bool   `long:"mono"`
	ConfirmExit    bool   `long:"confirm-exit-on-error"`
	NonInteractive bool   `short:"n" long:"non-interactive"`
}

// DisableColor returns whether color output should be disabled
// By default it only returns false if stdout is a terminal.  This check can be disabled with
// the checkTerminal flag
func (of outputFlags) DisableColor(checkTerminalFlag ...bool) bool {
	checkTerminal := true
	if len(checkTerminalFlag) > 0 {
		checkTerminal = checkTerminalFlag[0]
	}
	_, noColorEnv := os.LookupEnv("NO_COLOR")
	return of.Mono || noColorEnv || (checkTerminal && !terminal.StdoutSupportsColors())
}

func parseOutputFlags(args []string) outputFlags {
	var flagSet outputFlags
	parser := flags.NewParser(&flagSet, flags.IgnoreUnknown)
	_, err := parser.ParseArgs(args)
	if err != nil {
		logging.Warningf("Could not parse output flag: %s", err.Error())
	}

	return flagSet
}

func initOutput(flags outputFlags, formatName string) (output.Outputer, error) {
	if formatName == "" {
		formatName = flags.Output
	}

	out, err := output.New(formatName, &output.Config{
		OutWriter:   os.Stdout,
		ErrWriter:   os.Stderr,
		Colored:     !flags.DisableColor(),
		Interactive: true,
	})
	if err != nil {
		if errors.Is(err, output.ErrNotRecognized) {
			// The formatter might still be registered, so default to plain for now
			logging.Warningf("Output format not recognized: %s, defaulting to plain output instead", formatName)
			return initOutput(flags, string(output.PlainFormatName))
		}
		logging.Errorf("Could not create outputer, name: %s, error: %s", formatName, err.Error())
		rollbar.Error("Could not create outputer, name: %s, error: %s", formatName, err.Error())
		return nil, errs.Wrap(err, "output.New %s failed", formatName)
	}
	return out, nil
}

// setPrinterColors disables colored output in the printer packages in case the
// terminal does not support it, or if requested by the output arguments
func setPrinterColors(flags outputFlags) {
	disableColor := flags.DisableColor()
	survey.DisableColor = disableColor
}
