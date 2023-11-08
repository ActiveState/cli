package request

import "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"

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
		"strategy": model.RevertCommitStrategyForce,
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
    ... on RevertConflict {
      __typename
      message
      commitId
      targetCommitId
      conflictPaths
    }
    ... on CommitHasNoParent {
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
      path
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
    }
    ... on NoChangeSinceLastCommit {
      message
      commitId
    }
  }
}`
}

func (r *revertCommit) Vars() map[string]interface{} {
	return r.vars
}
