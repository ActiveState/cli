package export

import (
	"github.com/ActiveState/cli/internal/captain"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
)

type Export struct{}

func NewExport() *Export {
	return &Export{}
}

func (e *Export) Run(cmd *captain.Command) error {
	logging.Debug("Execute")
	err := cmd.Usage()
	if err != nil {
		failures.Handle(err, locale.T("package_err_help"))
		return nil
	}
	return nil
}
