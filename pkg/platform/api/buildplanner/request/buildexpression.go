package request

func BuildExpression(owner, project, commitID string) *buildScriptByCommitID {
	return &buildScriptByCommitID{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"commitID":     commitID,
	}}
}

type buildScriptByCommitID struct {
	vars map[string]interface{}
}

func (b *buildScriptByCommitID) Query() string {
	return `
query ($organization: String!, $project: String!, $commitID: String!) {
  project(organization: $organization, project: $project) {
    ... on Project {
      __typename
      commit(vcsRef: $commitID) {
        ... on Commit {
          __typename
          script
        }
        ... on NotFound {
          __typename
          message
        }
      }
    }
    ... on NotFound {
      __typename
      message
    }
  }
}
`
}

func (b *buildScriptByCommitID) Vars() map[string]interface{} {
	return b.vars
}
