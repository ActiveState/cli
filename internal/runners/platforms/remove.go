package platforms

import (
	"github.com/ActiveState/cli/internal/logging"
)

// Remove manages the removeing execution context.
type Remove struct {
	printer Printer
}

// NewRemove prepares a remove execution context for use.
func NewRemove(p Printer) *Remove {
	return &Remove{
		printer: p,
	}
}

// Run executes the remove behavior.
func (r *Remove) Run() error {
	logging.Debug("Execute platforms remove")

	return remove(r.printer)
}

func remove(printer Printer) error {
	return nil
}
