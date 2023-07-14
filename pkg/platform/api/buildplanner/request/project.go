package request

import "github.com/ActiveState/cli/internal/logging"

func BuildPlanByProject(organization, project, commitID string) *buildPlanByProject {
	logging.Debug("BuildPlanByProject")
	bp := &buildPlanByProject{map[string]interface{}{
		"organization": organization,
		"project":      project,
		"commitID":     commitID,
	}}

	return bp
}

type buildPlanByProject struct {
	vars map[string]interface{}
}

func (b *buildPlanByProject) Query() string {
	return `
query ($commitID: String!, $organization: String!, $project: String!) {
  project(organization: $organization, project: $project) {
    ... on Project {
      commit(vcsRef: $commitID) {
        ... on Commit {
          __typename
          expr
          build {
            __typename
            ... on BuildCompleted {
              buildLogIds {
                ... on AltBuildId {
                  id
                }
              }
            }
            ... on BuildStarted {
              buildLogIds {
                ... on AltBuildId {
                  id
                }
              }
            }
            ... on Build {
              status
              terminals {
                tag
                nodeIds
              }
              sources: nodes {
                ... on Source {
                  nodeId
                  name
                  namespace
                  version
                }
              }
              steps: steps {
                ... on Step {
                  stepId
                  inputs {
                    tag
                    nodeIds
                  }
                  outputs
                }
              }
              artifacts: nodes {
                ... on ArtifactSucceeded {
                  __typename
                  nodeId
                  mimeType
                  generatedBy
                  runtimeDependencies
                  status
                  logURL
                  url
                  checksum
                }
                ... on ArtifactUnbuilt {
                  __typename
                  nodeId
                  mimeType
                  generatedBy
                  runtimeDependencies
                  status
                }
                ... on ArtifactStarted {
                  __typename
                  nodeId
                  mimeType
                  generatedBy
                  runtimeDependencies
                  status
                }
                ... on ArtifactTransientlyFailed {
                  __typename
                  nodeId
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
                  __typename
                  nodeId
                  mimeType
                  generatedBy
                  runtimeDependencies
                  status
                  logURL
                  errors
                }
              }
            }
            ... on Error {
              message
            }
            ... on PlanningError {
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
        ... on Error {
          message
        }
        ... on NotFound {
          message
        }
      }
    }
    ... on Error{
      message
    }
    ... on NotFound {
      message
    }
  }
}
`
}

func (b *buildPlanByProject) Vars() map[string]interface{} {
	return b.vars
}
