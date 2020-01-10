package activate

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type Activate struct {
	namespaceSelect  namespaceSelectAble
	activateCheckout CheckoutAble
}

type ActivateParams struct {
	Namespace     string
	PreferredPath string
	Output        commands.Output
}

func NewActivate(namespaceSelect namespaceSelectAble, activateCheckout CheckoutAble) *Activate {
	return &Activate{
		namespaceSelect,
		activateCheckout,
	}
}

func (r *Activate) Run(params *ActivateParams) error {
	return r.run(params, activationLoop)
}

func sendProjectIDToAnalytics(namespace string, configFile string) {
	names, fail := project.ParseNamespaceOrConfigfile(namespace, configFile)
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

	targetPath, err := r.setupPath(params.Namespace, params.PreferredPath)
	if err != nil {
		if params.Namespace == "" {
			return failures.FailUserInput.Wrap(err)
		}
		err := r.activateCheckout.Run(params.Namespace, targetPath)
		if err != nil {
			return err
		}
	}

	if params.Output != "" {
		return activateOutput(targetPath, params.Output)
	}

	go sendProjectIDToAnalytics(params.Namespace, filepath.Join(targetPath, constants.ConfigFileName))

	return activatorLoop(targetPath, activate)
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
		targetPath, err = os.Getwd()
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
