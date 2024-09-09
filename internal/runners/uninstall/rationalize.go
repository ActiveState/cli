package uninstall

import (
	"errors"
	"fmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/rationalizers"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
)

func (u *Uninstall) rationalizeError(rerr *error) {
	var noMatchesErr *errNoMatches

	switch {
	case rerr == nil:
		return

	// Error staging a commit during uninstall.
	case errors.As(*rerr, &noMatchesErr):
		pkgs := []string{}
		for _, pkg := range noMatchesErr.packages {
			name := pkg.Name
			if pkg.Namespace != "" {
				name = fmt.Sprintf("%s/%s", pkg.Namespace, pkg.Name)
			}
			pkgs = append(pkgs, fmt.Sprintf("[ACTIONABLE]%s[/RESET]", name))
		}
		*rerr = errs.WrapUserFacing(*rerr, locale.Tr("err_uninstall_nomatch", noMatchesErr.packages.String()))

	// Error staging a commit during install.
	case errors.As(*rerr, ptr.To(&bpResp.CommitError{})):
		rationalizers.HandleCommitErrors(rerr)

	}
}
