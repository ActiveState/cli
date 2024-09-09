package buildplanner

import (
	"os"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/request"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
	"github.com/ActiveState/cli/pkg/projectfile"
)

func (b *BuildPlanner) GetBuildScript(owner, project, commitID string) (*buildscript.BuildScript, error) {
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

	checkoutInfo := &buildscript.CheckoutInfo{
		Project: projectURL(owner, project, commitID),
		AtTime:  time.Time(resp.Commit.AtTime),
	}
	script, err := buildscript.UnmarshalBuildExpression(resp.Commit.Expression, checkoutInfo)
	if err != nil {
		return nil, errs.Wrap(err, "failed to parse build expression")
	}

	return script, nil
}

func projectURL(owner, project, commitID string) string {
	// Note: cannot use api.GetPlatformURL() due to import cycle.
	host := constants.DefaultAPIHost
	if hostOverride := os.Getenv(constants.APIHostEnvVarName); hostOverride != "" {
		host = hostOverride
	}
	pjf := projectfile.NewProjectField()
	err := pjf.LoadProject("https://" + host)
	if err != nil {
		multilog.Error("could not initialize new project field: %v", err)
		return ""
	}
	pjf.SetNamespace(owner, project)
	pjf.SetLegacyCommitID(commitID)
	return pjf.String()
}
