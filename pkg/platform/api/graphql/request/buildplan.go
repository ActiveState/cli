package request

import "fmt"

func BuildPlan(owner, project, commitID string) *buildPlanByCommitID {
	return &buildPlanByCommitID{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"commitID":     commitID,
	}}
}

type buildPlanByCommitID struct {
	vars map[string]interface{}
}

func (b *buildPlanByCommitID) Query() string {
	return fmt.Sprintf(`
query ($organization: String!, $project: String!, $commitID: String!) {
  project(organization: $organization, project: $project) {
    ... on Project {
      __typename
      commit(vcsRef: $commitID) {
        ... on Commit {
          __typename
          graph
          %s
        }
        ... on CommitNotFound {
          __typename
          message
        }
      }
    }
    ... on ProjectNotFound {
      __typename
      message
    }
  }
}
`, buildResultFragment)
}

func (b *buildPlanByCommitID) Vars() map[string]interface{} {
	return b.vars
}
