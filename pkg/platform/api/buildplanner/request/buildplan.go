package request

import (
	"github.com/ActiveState/cli/internal/gqlclient"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
)

func BuildPlan(commitID, organization, project string, target *string) gqlclient.Request {
	if organization == "" && project == "" {
		return BuildPlanByCommitID(commitID, ptr.From(target, ""))
	}

	return BuildPlanByProject(organization, project, commitID, ptr.From(target, ""))
}

func BuildPlanTarget(commitID, organization, project, target string) gqlclient.Request {
	return BuildPlanByProject(organization, project, commitID, target)
}
