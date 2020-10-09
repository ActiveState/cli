package activate

import (
	"fmt"
	"path/filepath"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/primer"
	"github.com/ActiveState/cli/internal/subshell"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/ActiveState/cli/pkg/projectfile"
)

type Activate struct {
	namespaceSelect  namespaceSelectAble
	activateCheckout CheckoutAble
	out              output.Outputer
	proj             *project.Project
	subshell         subshell.SubShell
}

type ActivateParams struct {
	Namespace     *project.Namespaced
	PreferredPath string
	Command       string
	Replace       bool
}

type primeable interface {
	primer.Outputer
	primer.Projecter
	primer.Subsheller
}

func NewActivate(prime primeable, namespaceSelect namespaceSelectAble, activateCheckout CheckoutAble) *Activate {
	return &Activate{
		namespaceSelect,
		activateCheckout,
		prime.Output(),
		prime.Project(),
		prime.Subshell(),
	}
}

func (r *Activate) Run(params *ActivateParams) error {
	return r.run(params, activationLoop)
}

func (r *Activate) run(params *ActivateParams, activatorLoop activationLoopFunc) error {
	logging.Debug("Activate %v, %v", params.Namespace, params.PreferredPath)

	nSpace := params.Namespace.String()
	path := params.PreferredPath
	if params.Replace {
		nSpace = ""
		path = ""
	}

	pathToUse, err := r.pathToUse(nSpace, path)
	if err != nil {
		return locale.WrapError(err, "err_activate_pathtouse", "Could not figure out what path to use.")
	}

	projectToUse, err := r.projectToUse(pathToUse)
	if err != nil {
		return locale.WrapError(err, "err_activate_projecttouse", "Could not figure out what project to use.")
	}

	// Run checkout if no project was given
	if projectToUse == nil {
		if !params.Namespace.IsValid() {
			return locale.WrapError(err, "err_namespace_invalid", "Invalid namespace: {{.V0}}.", params.Namespace.String())
		}

		err := r.activateCheckout.Run(params.Namespace.String(), pathToUse)
		if err != nil {
			return err
		}

		var fail *failures.Failure
		projectToUse, fail = project.FromPath(pathToUse)
		if fail != nil {
			return locale.WrapError(fail, "err_activate_projectfrompath", "Something went wrong while creating project files.")
		}
	}

	projectPath := filepath.Dir(projectToUse.Source().Path())
	names := params.Namespace
	if !params.Replace {
		var fail *failures.Failure
		names, fail = project.ParseNamespaceOrConfigfile(names.String(), filepath.Join(projectPath, constants.ConfigFileName))
		if fail != nil {
			names = &project.Namespaced{}
			logging.Debug("error resolving namespace: %v", fail.ToError())
		}
	}
	// Send google analytics event with label set to project namespace
	analytics.EventWithLabel(analytics.CatRunCmd, "activate", names.String())

	// on --replace, replace namespace and commit id in as.yaml
	if params.Replace {
		updateProjectFile(projectToUse.Source(), names)
	}

	// If we're not using plain output then we should just dump the environment information
	if r.out.Type() != output.PlainFormatName {
		venv := virtualenvironment.Get()
		if fail := venv.Activate(); fail != nil {
			return locale.WrapError(fail.ToError(), "error_could_not_activate_venv", "Could not activate project. If this is a private project ensure that you are authenticated.")
		}
		env, err := venv.GetEnv(false, projectPath)
		if err != nil {
			return locale.WrapError(err, "err_activate_getenv", "Could not build environment for your runtime environment.")
		}
		if r.out.Type() == output.EditorV0FormatName {
			fmt.Println("[activated-JSON]")
		}
		r.out.Print(env)
		return nil
	}

	if params.Command != "" {
		r.subshell.SetActivateCommand(params.Command)
	}

	return activatorLoop(r.out, r.subshell, projectPath, activate)
}

func updateProjectFile(prjFile *projectfile.Project, names *project.Namespaced) error {
	if names.CommitID == nil || *names.CommitID == "" {
		latestID, fail := model.LatestCommitID(names.Owner, names.Project)
		if fail != nil {
			return locale.WrapInputError(fail.ToError(), "err_set_namespace_retrieve_commit", "Could not retrieve the latest commit for the specified project {{.V0}}.", names.String())
		}
		names.CommitID = latestID
	}

	err := prjFile.SetNamespace(names.String())
	if err != nil {
		return locale.WrapError(err, "err_activate_replace_write_namespace", "Failed to write new namespace to activestate.yaml.")
	}
	fail := prjFile.SetCommit(names.CommitID.String())
	if fail != nil {
		return locale.WrapError(fail.ToError(), "err_activate_replace_write_commit", "Failed to write commitID to activestate.yaml.")
	}
	return nil
}

func (r *Activate) pathToUse(namespace string, preferredPath string) (string, error) {
	switch {
	case namespace != "":
		// Checkout via namespace (eg. state activate org/project) and set resulting path
		return r.namespaceSelect.Run(namespace, preferredPath)
	case preferredPath != "":
		// Use the user provided path
		return preferredPath, nil
	default:
		// Get path from working directory
		targetPath, fail := projectfile.GetProjectFilePath()
		return filepath.Dir(targetPath), fail.ToError()
	}
}

func (r *Activate) projectToUse(path string) (*project.Project, error) {
	projectToUse, fail := project.FromPath(path)
	if fail != nil && !fail.Type.Matches(projectfile.FailNoProject) {
		return nil, locale.WrapError(fail, "err_activate_projectpath", "Could not find a valid project path.")
	}
	return projectToUse, nil
}
