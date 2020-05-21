package main

import (
	"os"

	"github.com/ActiveState/cli/cmd/state/internal/cmdtree"
	"github.com/ActiveState/cli/internal/config" // MUST be first!
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/deprecation"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/profile"
	"github.com/ActiveState/cli/internal/prompt"
	_ "github.com/ActiveState/cli/internal/prompt" // Sets up survey defaults
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// FailMainPanic is a failure due to a panic occuring while runnig the main function
var FailMainPanic = failures.Type("main.fail.panic", failures.FailUser)

func main() {
	// Set up logging
	logging.SetupRollbar()

	// Handle panics gracefully
	defer handlePanics(os.Exit)

	// Set up our output formatter/writer
	outFlags := parseOutputFlags(os.Args)
	out, fail := initOutput(outFlags, "")
	if fail != nil {
		os.Stderr.WriteString(locale.Tr("err_main_outputer", fail.Error()))
		os.Exit(1)
	}

	// Set up our legacy outputer
	setPrinterColors(outFlags)

	// Run our main command logic, which is logic that defers to the error handling logic below
	code, err := run(os.Args, out)
	if err != nil {
		out.Error(err)
	}

	os.Exit(code)
}

func run(args []string, out output.Outputer) (int, error) {
	// Set up profiling
	if os.Getenv(constants.CPUProfileEnvVarName) != "" {
		cleanup, err := profile.CPU()
		if err != nil {
			return 1, err
		}
		defer cleanup()
	}

	logging.Debug("ConfigPath: %s", config.ConfigPath())
	logging.Debug("CachePath: %s", config.CachePath())

	// Ensure any config set is preserved
	defer config.Save()

	// Retrieve project file
	pjPath, fail := projectfile.GetProjectFilePath()
	if fail != nil && fail.Type.Matches(projectfile.FailNoProjectFromEnv) {
		// Fail if we are meant to inherit the projectfile from the environment, but the file doesn't exist
		return 1, fail
	}

	// Auto update to latest state tool version, only runs once per day
	if updated, code, err := autoUpdate(args, out, pjPath); err != nil || updated {
		return code, err
	}

	// Set up project (if we have a valid path)
	var pj *project.Project
	if pjPath != "" {
		pjf, fail := projectfile.FromPath(pjPath)
		if fail != nil {
			return 1, fail
		}
		pj, fail = project.New(pjf)
		if fail != nil {
			return 1, fail
		}
	}

	// Forward call to specific state tool version, if warranted
	if code, fail := forwardIfWarranted(args, out, pj); fail != nil {
		return code, fail
	}

	// Check for deprecation
	deprecated, fail := deprecation.Check()
	if fail != nil {
		logging.Error("Could not check for deprecation: %s", fail.Error())
	}
	if deprecated != nil {
		date := deprecated.Date.Format(constants.DateFormatUser)
		if !deprecated.DateReached {
			out.Print(locale.Tr("warn_deprecation", date, deprecated.Reason))
		} else {
			out.Error(locale.Tr("err_deprecation", date, deprecated.Reason))
		}
	}

	// Run the actual command
	cmds := cmdtree.New(pj, out, prompt.New())
	err := cmds.Execute(args[1:])

	return unwrapError(err)
}
