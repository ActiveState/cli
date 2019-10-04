// +build !external

package main

import (
	"fmt"
	"strings"
	"time"

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

func (c *StateCommand) Children() []captain.Commander {
	return []captain.Commander{
		activate.Command.GetCobraCmd(),
	}
}
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

func runCPUProfiling() (cleanUp func(), fail *failures.Failure) {
	timeString := time.Now().Format("20060102-150405.000")
	timeString = strings.Replace(timeString, ".", "-", 1)
	cpuProfFile := fmt.Sprintf("cpu_%s.prof", timeString)

	cleanUpCPU, fail := profile.CPU(cpuProfFile)
	if fail != nil {
		return nil, fail
	}

	logging.Debug(fmt.Sprintf("profiling cpu (%s)", cpuProfFile))

	cleanUp = func() {
		logging.Debug("cleaning up cpu profiling")
		cleanUpCPU()
	}

	return cleanUp, nil
}
