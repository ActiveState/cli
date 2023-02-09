package request

func BuildPlan(owner, project, commitID string) *buildPlanByCommitID {
	return &buildPlanByCommitID{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"commitID":     commitID,
	}}
}

type buildPlanByCommitID struct {
	vars map[string]interface{}
}

func (b *buildPlanByCommitID) Query() string {
	return `
query ($organization: String!, $project: String!, $commitID: String!) {
  project(organization: $organization, project: $project) {
    ... on Project {
      __typename
      commit(vcsRef: $commitID) {
        ... on Commit {
          __typename
          build {
            __typename
            ... on Build {
              status
              terminals {
                tag
                targetIDs
              }
              sources: targets {
                ... on Source {
                  targetID
                  name
                  namespace
                  version
                }
              }
              steps: targets {
                ... on Step {
                  targetID
                  inputs {
                    tag
                    targetIDs
                  }
                  outputs
                }
              }
              artifacts: targets {
                __typename
                targetID
                ... on ArtifactSucceeded {
                  mimeType
                  generatedBy
                  runtimeDependencies
                  status
                  logURL
                  url
                  checksum
                }
                ... on ArtifactUnbuilt {
                  mimeType
                  generatedBy
                  runtimeDependencies
                  status
                }
                ... on ArtifactBuilding {
                  mimeType
                  generatedBy
                  runtimeDependencies
                  status
                }
                ... on ArtifactTransientlyFailed {
                  mimeType
                  generatedBy
                  runtimeDependencies
                  status
                  logURL
                  errors
                  attempts
                  nextAttemptAt
                }
                ... on ArtifactPermanentlyFailed {
                  mimeType
                  generatedBy
                  runtimeDependencies
                  status
                  logURL
                  errors
                }
              }
            }
            ... on PlanningError {
              subErrors {
                __typename
                ... on GenericSolveError {
                  path
                  message
                  isTransient
                  validationErrors {
                    jsonPath
                  }
                }
                ... on RemediableSolveError {
                  path
                  message
                  isTransient
                  errorType
                  validationErrors {
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
          __typename
          message
        }
      }
    }
    ... on NotFound {
      __typename
      message
    }
  }
}
`
}

func (b *buildPlanByCommitID) Vars() map[string]interface{} {
	return b.vars
}
