package request

import "github.com/ActiveState/cli/internal/gqlclient"

func BuildScript(owner, project, commitID string) *buildScriptByCommitID {
	return &buildScriptByCommitID{vars: map[string]interface{}{
		"organization": owner,
		"project":      project,
		"commitID":     commitID,
	}}
}

type buildScriptByCommitID struct {
	gqlclient.RequestBase
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
