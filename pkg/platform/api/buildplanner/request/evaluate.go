package request

import "github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"

func Evaluate(owner, project, target string, expression *buildexpression.BuildExpression) *evaluate {
	return &evaluate{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"target":       target,
		"expr":         expression,
	}}
}

type evaluate struct {
	vars map[string]interface{}
}

func (b *evaluate) Query() string {
	return `
query ($organization: String!, $project: String!, $expr: BuildExpr!) {
  project(organization: $organization, project: $project) {
    ... on Project {
      __typename
      name
      description
      evaluate(expr: $expr) {
        ... on Build {
          __typename
          status
        }
        ... on ParseError {
          __typename
          message
          subErrors {
            __typename
            message
            buildExprPath
          }
        }
        ... on ValidationError {
          __typename
          message
          subErrors {
            __typename
            message
            buildExprPath
          }
        }
        ... on PlanningError {
          __typename
          message
          subErrors {
            __typename
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
    ... on NotFound {
      type
      message
      resource
      mayNeedAuthentication
    }
  }
}`
}

func (b *evaluate) Vars() (map[string]interface{}, error) {
	return b.vars, nil
}
