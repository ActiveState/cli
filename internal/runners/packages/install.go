package packages

import (
	"fmt"
	"os"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/machineid"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
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
	if a.proj == nil {
		return a.addNoProject(params.Package.Name(), params.Package.Version())
		// return locale.NewInputError("err_no_project")
	}

	language, err := model.LanguageForCommit(a.proj.CommitUUID())
	if err != nil {
		return locale.WrapError(err, "err_fetch_languages")
	}

	ns := model.NewNamespacePkgOrBundle(language, nstype)

	return executePackageOperation(a.proj, a.cfg, a.out, a.auth, a.Prompter, params.Package.Name(), params.Package.Version(), model.OperationAdded, ns)
}

func (a *Install) addNoProject(name, version string) error {
	// Try with namespace to get a build working
	ns := model.NewNamespacePackage("python")
	commitID, err := model.CommitNoProject(model.HostPlatform, name, version, machineid.UniqID(), ns)
	fmt.Println("err:", err)
	if err != nil {
		return locale.WrapError(err, locale.Tl("err_commit_no_project", "Could not create commit without project"))
	}

	target, err := os.Getwd()
	if err != nil {
		return locale.WrapError(err, "err_add_get_wd", "Could not get working directory for new  project")
	}

	params := &projectfile.CreateParams{
		CommitID:   &commitID,
		ProjectURL: fmt.Sprintf("https://%s/commit/%s", constants.PlatformURL, commitID.String()),
		Directory:  target,
	}

	err = projectfile.Create(params)
	if err != nil {
		return locale.WrapError(err, "err_add_create_projectfile", "Could not create new projectfile")
	}

	return nil
}
