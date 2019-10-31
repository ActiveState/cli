package activate

import (
	"os"

	"github.com/ActiveState/cli/internal/logging"
)

type Activate struct {
	namespaceSelect namespaceSelectAble
}

func NewActivate(namespaceSelect namespaceSelectAble) *Activate {
	return &Activate{
		namespaceSelect,
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
