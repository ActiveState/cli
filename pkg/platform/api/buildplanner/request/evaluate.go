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
query ($organization: String!, $project: String!, $commitId: String!, $target: String!) {
  project(organization: $organization, project: $project) {
    ... on Project {
      __typename
      commit(vcsRef: $commitId) {
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
