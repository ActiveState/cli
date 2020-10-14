package packages

import (
	"github.com/ActiveState/cli/internal/headless"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// RemoveRunParams tracks the info required for running Remove.
type RemoveRunParams struct {
	Name string
}

// Remove manages the removing execution context.
type Remove struct {
	out  output.Outputer
	proj *project.Project
	prompt.Prompter
	auth *authentication.Auth
}

// NewRemove prepares a removal execution context for use.
func NewRemove(prime primeable) *Remove {
	return &Remove{
		prime.Output(),
		prime.Project(),
		prime.Prompt(),
		prime.Auth(),
	}
}

// Run executes the remove behavior.
func (r *Remove) Run(params RemoveRunParams) error {
	logging.Debug("ExecuteRemove")
	err := r.run(params)
	headless.Notify(r.out, r.proj, err, "packages")
	return err
}

func (r *Remove) run(params RemoveRunParams) error {
	// Commit the package
	pj := project.Get()
	language, fail := model.DefaultLanguageNameForProject(pj.Owner(), pj.Name())
	if fail != nil {
		return locale.WrapError(fail, "err_fetch_languages")
	}

	return executePackageOperation(r.out, r.auth, r.Prompter, language, params.Name, "", model.OperationRemoved)
}
