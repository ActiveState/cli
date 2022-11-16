package request

import "fmt"

func SaveAndBuild(owner, project, parentCommit, branchRef string, graph buildGraphInput) *buildPlanBySaveAndBuild {
	return &buildPlanBySaveAndBuild{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"commitID":     parentCommit,
		"branchRef":    branchRef,
		"graph":        graph,
	}}
}

type buildPlanBySaveAndBuild struct {
	vars map[string]interface{}
}

func (b *buildPlanBySaveAndBuild) Query() string {
	return fmt.Sprintf(`
mutation ($organization: String!, $project: String!, $parentCommit: String!, graph: BuildGraphInput!, branchRef: String!) {
  build(organization: $organization, project: $project, parentCommit: $parentCommit, graph: $graph, branchRef: $branchRef) {
    ... on Commit {
      __typename
      %s
    }
    ... on CommitNotFound {
      __typename
      message
    }
  }
}
	`, buildResultFragment)
}

func (b *buildPlanBySaveAndBuild) Vars() map[string]interface{} {
	return b.vars
}
