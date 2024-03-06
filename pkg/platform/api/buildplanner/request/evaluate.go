package request

func Evaluate(commitId, target string) *evaluate {
	return &evaluate{map[string]interface{}{
		"commitId": commitId,
		"target":   target,
	}}
}

type evaluate struct {
	vars map[string]interface{}
}

func (b *evaluate) Query() string {
	return `
query ($commitId: ID!, $target: String!) {
  commit(commitId: $commitId) {
    ... on Commit {
      __typename
      build(target: $target) {
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
