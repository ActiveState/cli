// +build !external

package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	secretsapi "github.com/ActiveState/cli/pkg/platform/api/secrets"
	"github.com/ActiveState/cli/state/activate"
	"github.com/ActiveState/cli/state/auth"
	"github.com/ActiveState/cli/state/events"
	"github.com/ActiveState/cli/state/export"
	"github.com/ActiveState/cli/state/internal/profile"
	"github.com/ActiveState/cli/state/invite"
	"github.com/ActiveState/cli/state/keypair"
	"github.com/ActiveState/cli/state/organizations"
	pkg "github.com/ActiveState/cli/state/package"
	"github.com/ActiveState/cli/state/projects"
	"github.com/ActiveState/cli/state/pull"
	"github.com/ActiveState/cli/state/run"
	"github.com/ActiveState/cli/state/scripts"
	"github.com/ActiveState/cli/state/secrets"
	"github.com/ActiveState/cli/state/show"
	"github.com/ActiveState/cli/state/update"
)

// register will register any commands and expanders
func register() {
	logging.Debug("register")

	secretsapi.InitializeClient()

	Command.Append(activate.Command)
	Command.Append(events.Command)
	Command.Append(update.Command)
	Command.Append(auth.Command)
	Command.Append(organizations.Command)
	Command.Append(projects.Command)
	Command.Append(show.Command)
	Command.Append(run.Command)
	Command.Append(scripts.Command)
	Command.Append(pull.Command)
	Command.Append(export.Command)
	Command.Append(invite.Command)
	Command.Append(pkg.Command)

	Command.Append(secrets.NewCommand(secretsapi.Get()).Config())
	Command.Append(keypair.Command)
}

func runProfiling() (cleanUp func(), fail *failures.Failure) {
	cleanUpFuncs := []func(){
		func() { logging.Debug("cleaning up profiling") },
	}
	cleanUp = func() {
		for _, fn := range cleanUpFuncs {
			fn()
		}
	}

	if os.Getenv(constants.CPUProfileEnvVarName) != "" {
		cleanUpFunc, fail := runCPUProfiling()
		if fail != nil {
			cleanUp()
			return nil, fail
		}
		cleanUpFuncs = append(cleanUpFuncs, cleanUpFunc)
	}

	return cleanUp, nil
}

func runCPUProfiling() (cleanUp func(), fail *failures.Failure) {
	logging.Debug("profiling cpu")

	timeString := time.Now().Format("20060102-150405.000")
	timeString = strings.Replace(timeString, ".", "-", 1)
	cpuProfFile := fmt.Sprintf("cpu_%s.prof", timeString)

	return profile.CPU(cpuProfFile)
}
