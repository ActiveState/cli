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

func (c *createProject) Vars() (map[string]interface{}, error) {
	return c.vars, nil
}
