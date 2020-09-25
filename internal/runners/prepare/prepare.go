package prepare

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
)

type primeable interface {
	primer.Outputer
}

// Prepare manages the prepare execution context.
type Prepare struct {
	out output.Outputer
}

// New prepares a prepare execution context for use.
func New(prime primeable) *Prepare {
	return &Prepare{
		out: prime.Output(),
	}
}

// Run executes the prepare behavior.
func (r *Prepare) Run() error {
	logging.Debug("ExecutePrepare")

	binDir := filepath.Join(config.CachePath(), "prepareBin")
	fail := fileutils.Mkdir(binDir)
	if fail != nil {
		return fail.ToError()
	}

	r.out.Print(binDir)

	return nil
}
