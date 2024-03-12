package request

func BuildPlanByProject(organization, project, commitID, target string) *buildPlanByProject {
	bp := &buildPlanByProject{map[string]interface{}{
		"organization": organization,
		"project":      project,
		"commitID":     commitID,
		"target":       target,
	}}

	return bp
}

type buildPlanByProject struct {
	vars map[string]interface{}
}

func (b *buildPlanByProject) Query() string {
	return `
query ($commitID: String!, $organization: String!, $project: String!, $target: String) {
  project(organization: $organization, project: $project) {
    ... on Project {
      commit(vcsRef: $commitID) {
        ... on Commit {
          __typename
          expr
          commitId
          build(target: $target) {
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
                  displayName
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
                  displayName
                  mimeType
                  generatedBy
                  runtimeDependencies
                  status
                }
                ... on ArtifactStarted {
                  __typename
                  nodeId
                  displayName
                  mimeType
                  generatedBy
                  runtimeDependencies
                  status
                }
                ... on ArtifactTransientlyFailed {
                  __typename
                  nodeId
                  displayName
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
                  displayName
                  mimeType
                  generatedBy
                  runtimeDependencies
                  status
                  logURL
                  errors
                }
                ... on ArtifactFailed {
                  __typename
                  nodeId
                  displayName
                  mimeType
                  generatedBy
                  runtimeDependencies
                  status
                  logURL
                  errors
                }
              }
              resolvedRequirements {
                requirement {
                  name
                  namespace
                  version_requirements: versionRequirements {
                   	comparator
                    version
                  }
                }
                resolvedSource
              }
            }
            ... on Error {
              message
            }
            ... on PlanningError {
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
        ... on Error {
          message
        }
        ... on NotFound {
          __typename
          type
          resource
          mayNeedAuthentication
          message
        }
      }
    }
    ... on Error {
      __typename
      message
    }
    ... on NotFound {
      __typename
      type
      resource
      mayNeedAuthentication
      message
    }
  }
}
`
}

func (b *buildPlanByProject) Vars() (map[string]interface{}, error) {
	return b.vars, nil
}
