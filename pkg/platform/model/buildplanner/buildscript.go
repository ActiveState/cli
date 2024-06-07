package buildplanner

import (
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/request"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
)

func (b *BuildPlanner) GetBuildScript(commitID string) (*buildscript.BuildScript, error) {
	logging.Debug("GetBuildExpression, commitID: %s", commitID)
	resp := &bpResp.BuildExpressionResponse{}
	err := b.client.Run(request.BuildExpression(commitID), resp)
	if err != nil {
		return nil, processBuildPlannerError(err, "failed to fetch build expression")
	}

	if resp.Commit == nil {
		return nil, errs.New("Commit is nil")
	}

	if bpResp.IsErrorResponse(resp.Commit.Type) {
		return nil, bpResp.ProcessCommitError(resp.Commit, "Could not get build expression from commit")
	}

	if resp.Commit.Expression == nil {
		return nil, errs.New("Commit does not contain expression")
	}

	script, err := buildscript.UnmarshalBuildExpression(resp.Commit.Expression, ptr.To(time.Time(resp.Commit.AtTime)))
	if err != nil {
		return nil, errs.Wrap(err, "failed to parse build expression")
	}

	return script, nil
}
