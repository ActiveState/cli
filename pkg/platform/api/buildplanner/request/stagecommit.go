package request

import "github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"

func StageCommit(owner, project, parentCommit, description string, expression *buildexpression.BuildExpression) *buildPlanByStageCommit {
	return &buildPlanByStageCommit{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"parentCommit": parentCommit,
		"description":  description,
		"expr":         expression,
		"atTime":       "", // default to the latest timestamp
	}}
}

type buildPlanByStageCommit struct {
	vars map[string]interface{}
}

func (b *buildPlanByStageCommit) Query() string {
	return `
mutation ($organization: String!, $project: String!, $parentCommit: ID!, $description: String!, $expr: BuildExpr!) {
  stageCommit(
    input: {organization: $organization, project: $project, parentCommitId: $parentCommit, description: $description, expr: $expr}
  ) {
    ... on Commit {
      __typename
      expr
      commitId
    }
    ... on Error {
      __typename
      message
    }
    ... on NotFound {
      __typename
      message
      type
      resource
      mayNeedAuthentication
    }
    ... on ParseError {
      __typename
      message
      path
    }
    ... on Forbidden {
      __typename
      operation
      message
      resource
    }
    ... on HeadOnBranchMoved {
      __typename
      commitId
      branchId
      message
    }
    ... on NoChangeSinceLastCommit {
      __typename
      commitId
      message
    }
  }
}
`
}

func (b *buildPlanByStageCommit) Vars() (map[string]interface{}, error) {
	return b.vars, nil
}
