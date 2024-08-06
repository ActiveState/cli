package manifest

import (
	runtime_runbit "github.com/ActiveState/cli/internal/runbits/runtime"
	auth "github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/project"
)

func rationalizeError(proj *project.Project, auth *auth.Auth, rerr *error) {
	switch {
	case rerr == nil:
		return
	default:
		runtime_runbit.RationalizeSolveError(proj, auth, rerr)
		return
	}
}
