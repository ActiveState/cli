package request

func CreateProject(owner, project string, private bool, expr []byte, description string) *createProject {
	return &createProject{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"private":      private,
		"expr":         string(expr),
		"description":  description,
		"atTime":       "", // default to the latest timestamp
	}}
}

type createProject struct {
	vars map[string]interface{}
}

func (c *createProject) Query() string {
	return `
mutation ($organization: String!, $project: String!, $private: Boolean!, $expr: BuildExpr!, $description: String!) {
  createProject(
    input: {organization: $organization, project: $project, private: $private, expr: $expr, description: $description}
  ) {
    ... on ProjectCreated {
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

func (c *createProject) Vars() (map[string]interface{}, error) {
	return c.vars, nil
}
