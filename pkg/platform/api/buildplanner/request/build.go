package request

func Build(owner, project, commitId, target string) *build {
	return &build{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"commitId":     commitId,
		"target":       target,
	}}
}

type build struct {
	vars map[string]interface{}
}

func (b *build) Query() string {
	return `
mutation ($organization: String!, $project: String!, $commitId: String!, $target: String) {
  buildCommitTarget(
    input: {organization: $organization, project: $project, commitId: $commitId, target: $target}
  ) {
    ... on Build {
      __typename
      status
    }
    ... on Error {
      __typename
      message
    }
    ... on ErrorWithSubErrors {
      __typename
      subErrors {
        __typename
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

func (b *build) Vars() (map[string]interface{}, error) {
	return b.vars, nil
}
