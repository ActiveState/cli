package prepare

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/config"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
)

// Prepare manages the prepare execution context.
type Prepare struct {
	out output.Outputer
}

// New prepares a prepare execution context for use.
func New(out output.Outputer) *Prepare {
	return &Prepare{
		out: out,
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
