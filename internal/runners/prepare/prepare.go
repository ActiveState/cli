package prepare

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/fileutils"
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

	binDir := filepath.Join(config.CachePath(), "prepareBin")
	fail := fileutils.Mkdir(binDir)
	if fail != nil {
		return locale.WrapError(fail.ToError(), "err_prepare_bin_dir", "Could not create global installations directory")
	}

	envUpdates := map[string]string{
		"PATH": fmt.Sprintf("%s%s%s", binDir, string(os.PathListSeparator), os.Getenv("PATH")),
	}

	fail = r.subshell.WriteUserEnv(envUpdates, true)
	if fail != nil {
		return locale.WrapError(fail.ToError(), "err_prepare_update_env", "Could not update user environment")
	}

	if runtime.GOOS == "windows" {
		r.out.Print(locale.Tr("prepare_instructions_windows", binDir))
	} else {
		r.out.Print(locale.Tr("prepare_instructions_lin_mac", binDir))
	}

	return nil
}
