package runtime

import (
	"errors"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils"
	"github.com/ActiveState/cli/internal/runbits"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	rt "github.com/ActiveState/cli/pkg/platform/runtime"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type ErrUpdate struct {
	*locale.LocalizedError
}

type Configurable interface {
	GetString(key string) string
	GetBool(key string) bool
}

// NewFromProject is a helper function that creates a new runtime or updates an existing one for
// the given project.
func NewFromProject(
	proj *project.Project,
	customCommitID *strfmt.UUID,
	trigger target.Trigger,
	an analytics.Dispatcher,
	svcModel *model.SvcModel,
	out output.Outputer,
	auth *authentication.Auth,
	cfg Configurable) (_ *rt.Runtime, rerr error) {
	defer rationalizeError(auth, proj, &rerr)

	if proj.IsHeadless() {
		return nil, rationalize.ErrHeadless
	}

	rti, err := rt.New(target.NewProjectTarget(proj, customCommitID, trigger), an, svcModel, auth, cfg, out)
	if err == nil {
		return rti, nil
	}

	if errors.Is(err, rt.NeedsCommitError) {
		out.Notice(locale.T("notice_commit_build_script"))
	}

	if rti.NeedsUpdate() {
		pg := runbits.NewRuntimeProgressIndicator(out)
		defer rtutils.Closer(pg.Close, &rerr)
		if err := rti.SolveAndUpdate(pg); err != nil {
			return nil, &ErrUpdate{locale.WrapError(err, "err_update_runtime", "Could not update runtime installation.")}
		}
		return rti, nil
	}

	return rti, nil
}
