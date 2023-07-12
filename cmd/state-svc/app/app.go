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

	installRoot, err := installation.InstallRoot(dir)
	if err != nil {
		return nil, errs.Wrap(err, "Could not determine install root")
	}

	return app.New(constants.SvcAppName, svcExec, []string{"start"}, installRoot, Options)
}

func New() (*app.App, error) {
	svcExec, err := installation.ServiceExec()
	if err != nil {
		return nil, errs.Wrap(err, "Could not determine service executable")
	}

	installRoot, err := installation.InstallPathFromExecPath()
	if err != nil {
		return nil, errs.Wrap(err, "Could not determine install root from exec")
	}

	return app.New(constants.SvcAppName, svcExec, []string{"start"}, installRoot, Options)
}
