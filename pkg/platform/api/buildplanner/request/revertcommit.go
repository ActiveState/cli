package request

import (
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
)

func RevertCommit(organization, project, targetVcsRef, commitID string) *revertCommit {
	return &revertCommit{map[string]interface{}{
		"organization": organization,
		"project":      project,
		"targetVcsRef": targetVcsRef,
		"commitId":     commitID,
		// Currently, we use the force strategy for all revert commits.
		// This is because we don't have a way to show the user the conflicts
		// and let them resolve them yet.
		// https://activestatef.atlassian.net/browse/AR-80?focusedCommentId=46998
		"strategy": types.RevertCommitStrategyForce,
	}}
}

type revertCommit struct {
	vars map[string]interface{}
}

func (r *revertCommit) Query() string {
	return `
mutation ($organization: String!, $project: String!, $commitId: String!, $targetVcsRef: String!, $strategy: RevertStrategy) {
  revertCommit(
    input: {organization: $organization, project: $project, commitId: $commitId, targetVcsRef: $targetVcsRef, strategy: $strategy}
  ) {
    ... on RevertedCommit {
      __typename
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
        ... on RemediableError {
          possibleRemediations {
            description
            suggestedPriority
          }
        }
        ... on GenericSolveError {
          path
          message
          isTransient
          validationErrors {
            error
            jsonPath
          }
        }
        ... on RemediableSolveError {
          path
          message
          isTransient
          errorType
          validationErrors {
            error
            jsonPath
          }
          suggestedRemediations {
            remediationType
            command
            parameters
          }
        }
      }
    }
  }
}
`
}

func (r *revertCommit) Vars() (map[string]interface{}, error) {
	return r.vars, nil
}
