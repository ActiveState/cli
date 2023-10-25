package request

func AttachStagedCommit(owner, project, parentCommit, commit, branch string) *attachStagedCommit {
	return &attachStagedCommit{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"parentCommit": parentCommit,
		"commit":       commit,
		"branch":       branch,
	}}
}

type attachStagedCommit struct {
	vars map[string]interface{}
}

func (b *attachStagedCommit) Query() string {
	return `
mutation ($organization: String!, $project: String!, $parentCommit: ID!, $commit: ID!, $branch: String!) {
  attachStagedCommit(input:{organization:$organization, project:$project, parentCommitId:$parentCommit, stagedCommitId:$commit, branchRef:$branch}) {
    ... on Commit {
      __typename
      commitId
    }
    ... on NotFound {
      __typename
      message
      mayNeedAuthentication
    }
    ... on ParseError {
      __typename
      message
    }
    ... on Forbidden {
      __typename
      message
    }
    ... on HeadOnBranchMoved {
      __typename
      message
      commitId
      branchId
    }
    ... on NoChangeSinceLastCommit {
      __typename
      message
      commitId
    }
  }
}
`
}

func (b *attachStagedCommit) Vars() map[string]interface{} {
	return b.vars
}
