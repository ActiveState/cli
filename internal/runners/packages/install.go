package packages

import (
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

// InstallRunParams tracks the info required for running Install.
type InstallRunParams struct {
	Package PackageVersion
}

// Install manages the installing execution context.
type Install struct {
	cfg  configurable
	out  output.Outputer
	proj *project.Project
	prompt.Prompter
	auth *authentication.Auth
}

// NewInstall prepares an installation execution context for use.
func NewInstall(prime primeable) *Install {
	return &Install{
		prime.Config(),
		prime.Output(),
		prime.Project(),
		prime.Prompt(),
		prime.Auth(),
	}
}

// Run executes the install behavior.
func (a *Install) Run(params InstallRunParams, nstype model.NamespaceType) error {
	logging.Debug("ExecuteInstall")
	var language string
	var err error
	if a.proj == nil {
		lang, err := a.getPackageLanguage(params.Package.Name(), params.Package.Version())
		if err != nil {
			return locale.WrapError(err, "err_install_get_langauge", "Could not get language for package: {{.V0}}", params.Package.Name())
		}
		language = lang
	} else {
		language, err = model.LanguageForCommit(a.proj.CommitUUID())
		if err != nil {
			return locale.WrapError(err, "err_fetch_languages")
		}
	}

	ns := model.NewNamespacePkgOrBundle(language, nstype)

	// TODO: Update package operation function to handle a nil project
	// Maybe write this as it's own function first
	return executePackageOperation(a.proj, a.cfg, a.out, a.auth, a.Prompter, params.Package.Name(), params.Package.Version(), model.OperationAdded, ns)
}

func (a *Install) getPackageLanguage(name, version string) (string, error) {
	ns := model.NewBlankNamespace()
	packages, err := model.SearchIngredientsStrict(ns, name)
	if err != nil {
		return "", locale.WrapError(err, "package_ingredient_err_search", "Failed to resolve ingredient named: {{.V0}}", name)
	}

	if len(packages) == 0 {
		return "", errs.AddTips(
			locale.NewInputError("err_install_no_package", `No packages in our catalogue are an exact match for [NOTICE]"{{.V0}}"[/RESET].`, name),
			locale.Tl("info_try_search", "Valid package names can be searched using [ACTIONABLE]`state search {package_name}`[/RESET]"),
			locale.Tl("info_request", "Request a package at [ACTIONABLE]https://community.activestate.com/[/RESET]"),
		)
	}

	// TODO: Properly parse namespace
	data := strings.Split(*packages[0].Ingredient.PrimaryNamespace, "/")
	return data[1], nil
}
