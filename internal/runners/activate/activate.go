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
	"github.com/ActiveState/cli/pkg/project"
)

type Activate struct {
	namespaceSelect  namespaceSelectAble
	activateCheckout CheckoutAble
	out              output.Outputer
	subshell         subshell.SubShell
}

type ActivateParams struct {
	Namespace     *project.Namespaced
	PreferredPath string
	Command       string
}

type primeable interface {
	primer.Outputer
	primer.Subsheller
}

func NewActivate(prime primeable, namespaceSelect namespaceSelectAble, activateCheckout CheckoutAble) *Activate {
	return &Activate{
		namespaceSelect,
		activateCheckout,
		prime.Output(),
		prime.Subshell(),
	}
}

func (r *Activate) Run(params *ActivateParams) error {
	return r.run(params, activationLoop)
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

	// Send google analytics event with label set to project namespace
	names, fail := project.ParseNamespaceOrConfigfile(params.Namespace.String(), filepath.Join(targetPath, constants.ConfigFileName))
	if fail != nil {
		names = &project.Namespaced{}
		logging.Debug("error resolving namespace: %v", fail.ToError())
	}
	analytics.EventWithLabel(analytics.CatRunCmd, "activate", names.String())

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

	if params.Command != "" {
		r.subshell.SetActivateCommand(params.Command)
	}

	return activatorLoop(r.out, r.subshell, targetPath, activate)
}

func (r *Activate) setupPath(namespace string, preferredPath string) (string, error) {
	var (
		targetPath string
		proj       *project.Project
		fail       *failures.Failure
	)

	switch {
	// Checkout via namespace (eg. state activate org/project) and set resulting path
	case namespace != "":
		namesPath, err := r.namespaceSelect.Run(namespace, preferredPath)
		if err != nil {
			return "", err
		}
		proj, fail = project.FromPath(namesPath)
		targetPath = namesPath
	// Use the user provided path
	case preferredPath != "":
		proj, fail = project.FromPath(preferredPath)
		targetPath = preferredPath
	// Get path from working directory
	default:
		proj, fail = project.GetSafe()
	}
	if fail != nil {
		return targetPath, fail
	}

	return filepath.Dir(proj.Source().Path()), nil
}
