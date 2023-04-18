package app

import (
	"github.com/ActiveState/cli/internal/app"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/installation"
)

var Options = app.Options{}

func NewFromDir(dir string) (*app.App, error) {
	svcExec, err := installation.ServiceExecFromDir(dir)
	if err != nil {
		return nil, errs.Wrap(err, "Could not determine service executable")
	}

	return app.New(constants.SvcAppName, svcExec, []string{"start"}, Options)
}

func New() (*app.App, error) {
	svcExec, err := installation.ServiceExec()
	if err != nil {
		return nil, errs.Wrap(err, "Could not determine service executable")
	}

	return app.New(constants.SvcAppName, svcExec, []string{"start"}, Options)
}
