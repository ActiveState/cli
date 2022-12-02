package request

import (
	"fmt"

	model "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplan"
)

func SaveAndBuild(owner, project, parentCommit, branchRef, description string, graph *model.BuildGraph) *buildPlanBySaveAndBuild {
	return &buildPlanBySaveAndBuild{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"parentCommit": parentCommit,
		"branchRef":    branchRef,
		"description":  description,
		"graph":        graph,
	}}
}

type buildPlanBySaveAndBuild struct {
	vars map[string]interface{}
}

func (b *buildPlanBySaveAndBuild) Query() string {
	return fmt.Sprintf(`
mutation ($organization: String!, $project: String!, $parentCommit: String!, $graph: BuildGraph!, $branchRef: String!, $description:String!) {
  saveAndBuild(organization: $organization, project: $project, parentCommit: $parentCommit, graph: $graph, branchRef: $branchRef, description:$description) {
    ... on Commit {
      __typename
      graph
      commitId
      %s
    }
    ... on CommitNotFound {
      message
    }
    ... on BuildSubmissionError {
      message
    }
  }
}
`, buildResultFragment)
}

func (b *buildPlanBySaveAndBuild) Vars() map[string]interface{} {
	return b.vars
}
