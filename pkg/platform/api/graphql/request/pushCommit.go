package request

import (
	"fmt"

	model "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
)

func PushCommit(owner, project, parentCommit, branchRef, description string, script *model.BuildScript) *buildPlanByPushCommit {
	return &buildPlanByPushCommit{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"parentCommit": parentCommit,
		"branchRef":    branchRef,
		"description":  description,
		"script":       script,
	}}
}

type buildPlanByPushCommit struct {
	vars map[string]interface{}
}

func (b *buildPlanByPushCommit) Query() string {
	return fmt.Sprintf(`
mutation ($organization: String!, $project: String!, $parentCommit: String!, $script: BuildScript!, $branchRef: String!, $description:String!) {
  pushCommit(input:{org: $organization, project: $project, parentCommit: $parentCommit, script: $script, branchRef: $branchRef, description:$description}) {
    ... on Commit {
      __typename
      script
      commitId
      %s
    }
    ... on NotFound {
      message
    }
    ... on Error {
      message
    }
  }
}
`, buildResultFragment)
}

func (b *buildPlanByPushCommit) Vars() map[string]interface{} {
	return b.vars
}
