package activate

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

type Activate struct {
	namespaceSelect  namespaceSelectAble
	activateCheckout CheckoutAble
}

func NewActivate(namespaceSelect namespaceSelectAble, activateCheckout CheckoutAble) *Activate {
	return &Activate{
		namespaceSelect,
		activateCheckout,
	}
}

func (r *Activate) Run(namespace string, preferredPath string) error {
	return r.run(namespace, preferredPath, activationLoop)
}

func (r *Activate) run(namespace string, preferredPath string, activatorLoop activationLoopFunc) error {
	logging.Debug("Activate %v, %v", namespace, preferredPath)

	targetPath, err := r.setupPath(namespace, preferredPath)
	if err != nil {
		return err
	}

	// Checkout the project if it doesn't already exist at the target path
	configFile := filepath.Join(targetPath, constants.ConfigFileName)
	if !fileutils.FileExists(configFile) {
		if namespace == "" {
			return failures.FailUserInput.New("err_project_notexist_asyaml")
		}
		err := r.activateCheckout.Run(namespace, targetPath)
		if err != nil {
			return err
		}
	}

	return activatorLoop(targetPath, activate)
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
