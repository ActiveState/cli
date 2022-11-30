package request

func BuildGraph(owner, project, commitID string) *buildGraphByCommitID {
	return &buildGraphByCommitID{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"commitID":     commitID,
	}}
}

type buildGraphByCommitID struct {
	vars map[string]interface{}
}

func (b *buildGraphByCommitID) Query() string {
	return `
query ($organization: String!, $project: String!, $commitID: String!) {
  project(organization: $organization, project: $project) {
    ... on Project {
      __typename
      commit(vcsRef: $commitID) {
        ... on Commit {
          __typename
          graph
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
`
}

func (b *buildGraphByCommitID) Vars() map[string]interface{} {
	return b.vars
}
