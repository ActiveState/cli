package buildplanner

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/request"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
)

func buildScriptCommitInfo(owner, project, commitID string, atTime time.Time) *buildscript.CommitInfo {
	// Note: cannot use api.GetPlatformURL() due to import cycle.
	host := constants.DefaultAPIHost
	if hostOverride := os.Getenv(constants.APIHostEnvVarName); hostOverride != "" {
		host = hostOverride
	}
	u, err := url.Parse(fmt.Sprintf("https://%s/%s/%s", host, owner, project))
	if err != nil {
		multilog.Error("url parse for project URL failed: %w", err)
		return nil
	}
	q := u.Query()
	q.Set("commitID", commitID)
	u.RawQuery = q.Encode()
	projectURL := u.String()

	return &buildscript.CommitInfo{projectURL, atTime}
}

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

	script, err := buildscript.UnmarshalBuildExpression(resp.Commit.Expression, buildScriptCommitInfo("", "", "", time.Time(resp.Commit.AtTime)))
	if err != nil {
		return nil, errs.Wrap(err, "failed to parse build expression")
	}

	return script, nil
}
