package activate

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/print"
	"github.com/ActiveState/cli/pkg/cmdlets/commands"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/project"
)

type Activate struct {
	namespaceSelect  namespaceSelectAble
	activateCheckout CheckoutAble
	targetPath       string
}

type ActivateParams struct {
	Namespace     string
	PreferredPath string
	Output        commands.Output
}

func NewActivate(namespaceSelect namespaceSelectAble, activateCheckout CheckoutAble) *Activate {
	return &Activate{
		namespaceSelect:  namespaceSelect,
		activateCheckout: activateCheckout,
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

	var err error
	r.targetPath, err = r.setupPath(params.Namespace, params.PreferredPath)
	if err != nil {
		return err
	}

	configFile, err := r.setupConfigFile(r.targetPath, params)
	if err != nil {
		return err
	}

	switch params.Output {
	case commands.JSON, commands.EditorV0:
		err = os.Chdir(r.targetPath)
		if err != nil {
			return err
		}
		jsonString, err := envOutput(false)
		if err != nil {
			return err
		}
		print.Line("[activated-JSON]")
		print.Line(jsonString)
		return nil
	}

	go sendProjectIDToAnalytics(params.Namespace, configFile)

	return activatorLoop(r.targetPath, activate)
}

func (r *Activate) setupPath(namespace string, preferredPath string) (string, error) {
	switch {
	// Checkout via namespace (eg. state activate org/project) and set resulting path
	case namespace != "":
		return r.namespaceSelect.Run(namespace, preferredPath)
	// Use the user provided path
	case preferredPath != "":
		return preferredPath, nil
	// Get path from working directory
	default:
		return os.Getwd()
	}
}

func (r *Activate) setupConfigFile(targetPath string, params *ActivateParams) (string, error) {
	// Checkout the project if it doesn't already exist at the target path
	configFile := filepath.Join(targetPath, constants.ConfigFileName)
	if !fileutils.FileExists(configFile) {
		if params.Namespace == "" {
			proj, err := project.FromPath(targetPath)
			if err != nil {
				// The default failure returned by the project package is a big too vague,
				// we want to give the user something more actionable for the context they're in
				return "", failures.FailUserInput.New("err_project_notexist_asyaml")
			}
			logging.Debug("Updating namespace parameters to: %s", proj.Namespace())
			params.Namespace = proj.Namespace()
			r.targetPath = filepath.Dir(proj.ProjectFilePath())
		}
		err := r.activateCheckout.Run(params.Namespace, targetPath)
		if err != nil {
			return "", err
		}
	}

	return configFile, nil
}
