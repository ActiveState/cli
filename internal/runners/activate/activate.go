package activate

import (
	"fmt"
	"path/filepath"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/virtualenvironment"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type Activate struct {
	namespaceSelect  namespaceSelectAble
	activateCheckout CheckoutAble
	out              output.Outputer
}

type ActivateParams struct {
	Namespace     *project.Namespaced
	PreferredPath string
}

func NewActivate(out output.Outputer, namespaceSelect namespaceSelectAble, activateCheckout CheckoutAble) *Activate {
	return &Activate{
		namespaceSelect,
		activateCheckout,
		out,
	}
}

func (r *Activate) Run(params *ActivateParams) error {
	return r.run(params, activationLoop)
}

func sendProjectIDToAnalytics(namespace *project.Namespaced, configFile string) {
	names, fail := project.ParseNamespaceOrConfigfile(namespace.String(), configFile)
	if fail != nil {
		logging.Debug("error resolving namespace: %v", fail.ToError())
		return
	}

	platProject, fail := model.FetchProjectByName(names.Owner, names.Project)
	if fail != nil {
		logging.Debug("error getting platform project: %v", fail.ToError())
		return
	}
	projectID := platProject.ProjectID.String()
	analytics.EventWithLabel(
		analytics.CatBuild, analytics.ActBuildProject, projectID,
	)
}

func (r *Activate) run(params *ActivateParams, activatorLoop activationLoopFunc) error {
	logging.Debug("Activate %v, %v", params.Namespace, params.PreferredPath)

	targetPath, err := r.setupPath(params.Namespace.String(), params.PreferredPath)
	if err != nil {
		if !params.Namespace.IsValid() {
			return failures.FailUserInput.Wrap(err)
		}
		err := r.activateCheckout.Run(params.Namespace.String(), targetPath)
		if err != nil {
			return err
		}
	}

	go sendProjectIDToAnalytics(params.Namespace, filepath.Join(targetPath, constants.ConfigFileName))

	// If we're not using plain output then we should just dump the environment information
	if r.out.Type() != output.PlainFormatName {
		venv := virtualenvironment.Get()
		if fail := venv.Activate(); fail != nil {
			return locale.WrapError(fail.ToError(), "error_could_not_activate_venv", "Could not activate project. If this is a private project ensure that you are authenticated.")
		}
		env, err := venv.GetEnv(false, targetPath)
		if err != nil {
			return locale.WrapError(err, "err_activate_getenv", "Could not build environment for your runtime environment.")
		}
		if r.out.Type() == output.EditorV0FormatName {
			fmt.Println("[activated-JSON]")
		}
		r.out.Print(env)
		return nil
	}

	return activatorLoop(r.out, targetPath, activate)
}

func (r *Activate) setupPath(namespace string, preferredPath string) (string, error) {
	var (
		targetPath string
		err        error
	)

	switch {
	// Checkout via namespace (eg. state activate org/project) and set resulting path
	case namespace != "":
		targetPath, err = r.namespaceSelect.Run(namespace, preferredPath)
	// Use the user provided path
	case preferredPath != "":
		targetPath, err = preferredPath, nil
	// Get path from working directory
	default:
		targetPath, err = osutils.Getwd()
	}
	if err != nil {
		return "", err
	}

	proj, fail := project.FromPath(targetPath)
	if fail != nil {
		return targetPath, fail
	}

	return filepath.Dir(proj.Source().Path()), nil
}
