package request

import "github.com/ActiveState/cli/internal/gqlclient"

func BuildPlan(commitID, organization, project string) gqlclient.Request {
	if organization == "" && project == "" {
		return BuildPlanByCommitID(commitID)
	}

	return BuildPlanByProject(organization, project, commitID, "")
}

func BuildPlanTarget(commitID, organization, project, target string) gqlclient.Request {
	return BuildPlanByProject(organization, project, commitID, target)
}
