package request

func Evaluate(owner, project, commitId, target string) *evaluate {
	return &evaluate{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"commitId":     commitId,
		"target":       target,
	}}
}

type evaluate struct {
	vars map[string]interface{}
}

func (b *evaluate) Query() string {
	return `
mutation ($organization: String!, $project: String!, $commitId: String!, $target: String) {
  buildCommitTarget(
    input: {organization: $organization, project: $project, commitId: $commitId, target: $target}
  ) {
    ... on Build {
      __typename
      status
    }
    ... on PlanningError {
      __typename
      message
      subErrors {
        __typename
        ... on GenericSolveError {
          buildExprPath
          message
          isTransient
          validationErrors {
            error
            jsonPath
          }
        }
        ... on RemediableSolveError {
          buildExprPath
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
        ... on TargetNotFound {
          message
          requestedTarget
          possibleTargets
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
