package runtime

import (
	"errors"

	"github.com/ActiveState/cli/pkg/platform/runtime2/alternative"
	"github.com/ActiveState/cli/pkg/platform/runtime2/common"
	"github.com/ActiveState/cli/pkg/project"
)

// ErrNotInstalled is returned when the runtime is not locally installed yet.
// See the `setup.Setup` on how to set up a runtime installation.
var ErrNotInstalled = errors.New("Runtime not installed yet")

// IsNotInstalledError is a convenience function to checks if an error is NotInstalledError
func IsNotInstalledError(err error) bool {
	return errors.Is(err, ErrNotInstalled)
}

// NewRuntime creates a new runtime.
//
// The actual type is determined by the buildEngine file.
// Usually, this function is not called directly but via `setup.Setup.InstalledRuntime()`.
// If the runtime cannot be initialized a NotInstalledError is returned.
func NewRuntime(proj *project.Project) (common.Runtimer, error) {
	buildEngine := loadBuildEngine()
	if buildEngine == Alternative {
		return alternative.New(proj)
	}
	panic("Implement")
}

func loadBuildEngine() BuildEngine {
	panic("implement me")
}
