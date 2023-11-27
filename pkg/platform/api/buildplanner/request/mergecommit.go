package request

import "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"

func MergeCommit(owner, project, targetRef, otherRef string, strategy model.MergeStrategy) *mergeCommit {
	return &mergeCommit{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"targetRef":    targetRef,
		"otherRef":     otherRef,
		"strategy":     strategy,
	}}
}

type mergeCommit struct {
	vars map[string]interface{}
}

func (b *mergeCommit) Query() string {
	return `
mutation ($organization: String!, $project: String!, $targetRef: String!, $otherRef: String!, $strategy: MergeStrategy) {
  mergeCommit(input:{organization:$organization, project:$project, targetVcsRef:$targetRef, otherVcsRef:$otherRef, strategy:$strategy}) {
		... on MergedCommit {
			commit {
				__typename
				commitId
			}
		}
    ... on MergeConflict {
      __typename
      message
    }
    ... on FastForwardError {
      __typename
      message
    }
    ... on NoCommonBaseFound {
      __typename
      message
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
    ... on ValidationError {
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

func (b *mergeCommit) Vars() map[string]interface{} {
	return b.vars
}
