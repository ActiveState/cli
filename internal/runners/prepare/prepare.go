package prepare

import (
	"os"
	"path/filepath"

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

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	binDir := filepath.Join(wd, "bin")
	fail := fileutils.Mkdir(binDir)
	if fail != nil {
		return fail.ToError()
	}

	r.out.Print(binDir)

	return nil
}
