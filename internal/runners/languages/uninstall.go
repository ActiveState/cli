package languages

import (
	"github.com/ActiveState/cli/internal/language"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/runbits/requirements"
	"github.com/ActiveState/cli/pkg/platform/model"
)

// UninstallRunParams tracks the info required for running Uninstall.
type UninstallRunParams struct {
	Language string
}

// Uninstall manages the uninstalling execution context.
type Uninstall struct {
	prime primeable
}

// NewUninstall prepares an uninstallation execution context for use.
func NewUninstall(prime primeable) *Uninstall {
	return &Uninstall{prime}
}

// Run executes the uninstall behavior.
func (u *Uninstall) Run(params UninstallRunParams) error {
	logging.Debug("ExecuteLanguageUninstall")

	if u.prime.Project() == nil {
		return locale.NewInputError("err_no_project")
	}

	lang := language.MakeByName(params.Language)
	if !lang.Recognized() {
		return locale.NewInputError("error_unsupported_language", "", params.Language)
	}

	return requirements.NewRequirementOperation(u.prime).ExecuteRequirementOperation(
		lang.Requirement(),
		"",
		0,
		model.OperationRemoved,
		model.NamespaceLanguage,
	)
}
