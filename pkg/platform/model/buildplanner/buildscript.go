package buildplanner

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/request"
	bpResp "github.com/ActiveState/cli/pkg/platform/api/buildplanner/response"
)

func (b *BuildPlanner) GetBuildScript(commitID string) (*buildscript.BuildScript, error) {
	logging.Debug("GetBuildScript, commitID: %s", commitID)
	resp := &bpResp.Commit{}

	cacheKey := strings.Join([]string{"GetBuildScript", commitID}, "-")
	respRaw, err := b.cache.GetCache(cacheKey)
	if err != nil {
		return nil, errs.Wrap(err, "failed to get cache")
	}
	if respRaw != "" {
		if err := json.Unmarshal([]byte(respRaw), resp); err != nil {
			return nil, errs.Wrap(err, "failed to unmarshal cache: %s", cacheKey)
		}
	} else {
		err := b.client.Run(request.BuildExpression(commitID), resp)
		if err != nil {
			return nil, processBuildPlannerError(err, "failed to fetch build expression")
		}
		respBytes, err := json.Marshal(resp)
		if err != nil {
			return nil, errs.Wrap(err, "failed to marshal cache")
		}
		if err := b.cache.SetCache(cacheKey, string(respBytes), fetchCommitCacheExpiry); err != nil {
			return nil, errs.Wrap(err, "failed to set cache")
		}
	}

	if resp == nil {
		return nil, errs.New("Commit is nil")
	}

	if bpResp.IsErrorResponse(resp.Type) {
		return nil, bpResp.ProcessCommitError(resp, "Could not get build expression from commit")
	}

	if resp.Expression == nil {
		return nil, errs.New("Commit does not contain expression")
	}

	script := buildscript.New()
	script.SetAtTime(time.Time(resp.AtTime), true)
	if err := script.UnmarshalBuildExpression(resp.Expression); err != nil {
		return nil, errs.Wrap(err, "failed to parse build expression")
	}

	return script, nil
}
