package packages

import (
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// AddRunParams tracks the info required for running Add.
type AddRunParams struct {
	Name string
}

// Add manages the adding execution context.
type Add struct {
	out output.Outputer
}

// NewAdd prepares an addition execution context for use.
func NewAdd(out output.Outputer) *Add {
	return &Add{
		out: out,
	}
}

// Run executes the add behavior.
func (a *Add) Run(params AddRunParams) error {
	logging.Debug("ExecuteAdd")

	pj := project.Get()
	language, fail := model.DefaultLanguageForProject(pj.Owner(), pj.Name())
	if fail != nil {
		return fail.WithDescription("err_fetch_languages")
	}

	name, version := splitNameAndVersion(params.Name)

	return executeAddUpdate(a.out, language, name, version, model.OperationAdded)
}
