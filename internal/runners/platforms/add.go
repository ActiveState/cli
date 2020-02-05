package platforms

import (
	"github.com/ActiveState/cli/internal/logging"
)

// Add manages the adding execution context.
type Add struct {
	printer Printer
}

// NewAdd prepares an add execution context for use.
func NewAdd(p Printer) *Add {
	return &Add{
		printer: p,
	}
}

// Run executes the add behavior.
func (a *Add) Run() error {
	logging.Debug("Execute platforms add")

	return add(a.printer)
}

func add(printer Printer) error {
	return nil
}
