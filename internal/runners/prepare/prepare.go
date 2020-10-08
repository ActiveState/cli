package prepare

import (
	"runtime"

	"github.com/ActiveState/cli/internal/globaldefault"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/subshell"
)

type primeable interface {
	primer.Outputer
	primer.Subsheller
}

// Prepare manages the prepare execution context.
type Prepare struct {
	out      output.Outputer
	subshell subshell.SubShell
}

// New prepares a prepare execution context for use.
func New(prime primeable) *Prepare {
	return &Prepare{
		out:      prime.Output(),
		subshell: prime.Subshell(),
	}
}

// Run executes the prepare behavior.
func (r *Prepare) Run() error {
	logging.Debug("ExecutePrepare")

	if err := globaldefault.Prepare(r.subshell); err != nil {
		if runtime.GOOS != "linux" {
			return locale.WrapError(err, "err_prepare_update_env", "Could not prepare environment.")
		}
		logging.Debug("Encountered failure attempting to update user environment: %s", err)
		r.out.Notice(locale.T("prepare_env_warning"))
	}

	if runtime.GOOS == "windows" {
		r.out.Print(locale.Tr("prepare_instructions_windows", globaldefault.BinDir()))
	} else {
		r.out.Print(locale.Tr("prepare_instructions_lin_mac", globaldefault.BinDir()))
	}

	return nil
}
