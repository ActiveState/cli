package alternative

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/runtime/envdef"
)

// var _ runtime.EnvProvider = &Alternative{}

// Alternative is the specialization of a runtime for alternative builds
type Alternative struct {
	installPath string
}

// New is the constructor function for alternative runtimes
func New(installPath string) (*Alternative, error) {
	return &Alternative{installPath}, nil
}

// Environ returns the environment mapping
func (a *Alternative) Environ(inherit bool) (map[string]string, error) {
	mergedRuntimeDefinitionFile := filepath.Join(a.installPath, constants.RuntimeDefinitionFilename)
	rt, err := envdef.NewEnvironmentDefinition(mergedRuntimeDefinitionFile)
	if err != nil {
		return nil, locale.WrapError(
			err, "err_no_environment_definition",
			"Your installation seems corrupted.\nPlease try to re-run this command, as it may fix the problem.  If the problem persists, please report it in our forum: {{.V0}}",
			constants.ForumsURL,
		)
	}
	return rt.GetEnv(inherit), nil
}
