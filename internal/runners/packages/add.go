package packages

import (
	"github.com/ActiveState/cli/internal/headless"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// AddRunParams tracks the info required for running Add.
type AddRunParams struct {
	Name string
}

// Add manages the adding execution context.
type Add struct {
	out  output.Outputer
	proj *project.Project
	prompt.Prompter
	auth *authentication.Auth
}

// NewAdd prepares an addition execution context for use.
func NewAdd(prime primeable) *Add {
	return &Add{
		prime.Output(),
		prime.Project(),
		prime.Prompt(),
		prime.Auth(),
	}
}

// Run executes the add behavior.
func (a *Add) Run(params AddRunParams) error {
	err := a.run(params)
	headless.Notify(a.out, a.proj, err, "packages")
	return err
}

func (a *Add) run(params AddRunParams) error {
	logging.Debug("ExecuteAdd")

	language, fail := model.LanguageForCommit(a.proj.CommitUUID())
	if fail != nil {
		return fail.WithDescription("err_fetch_languages")
	}

	name, version := splitNameAndVersion(params.Name)

	return executePackageOperation(a.proj, a.out, a.auth, a.Prompter, language, name, version, model.OperationAdded)
}
