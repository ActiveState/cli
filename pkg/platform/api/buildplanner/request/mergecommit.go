package request

import (
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

func MergeCommit(owner, project, targetRef, otherRef string, strategy types.MergeStrategy) *mergeCommit {
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
      subErrors {
        message
        buildExprPath
      }
    }
    ... on ValidationError {
      __typename
      message
      subErrors {
        message
        buildExprPath
      }
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
    ... on InvalidInput {
      __typename
      message
    }
  }
}
`
}

func (b *mergeCommit) Vars() (map[string]interface{}, error) {
	return b.vars, nil
}
