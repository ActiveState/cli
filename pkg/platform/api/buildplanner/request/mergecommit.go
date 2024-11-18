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
  mergeCommit(
    input: {organization: $organization, project: $project, targetVcsRef: $targetRef, otherVcsRef: $otherRef, strategy: $strategy}
  ) {
    ... on MergedCommit {
      commit {
        __typename
        commitId
      }
    }
    ... on Error {
      __typename
      message
    }
    ... on ErrorWithSubErrors {
      __typename
      subErrors {
        __typename
        buildExprPath
        ... on GenericSolveError {
          message
          isTransient
          validationErrors {
            error
            jsonPath
          }
        }
        ... on RemediableSolveError {
          message
          isTransient
          errorType
          validationErrors {
            error
            jsonPath
          }
        }
      }
    }
  }
}
`
}

func (b *mergeCommit) Vars() (map[string]interface{}, error) {
	return b.vars, nil
}
