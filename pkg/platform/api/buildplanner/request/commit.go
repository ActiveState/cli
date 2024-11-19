package request

import "github.com/ActiveState/cli/internal/rtutils/ptr"

const TargetAll = "__all__"

func ProjectCommit(commitID, organization, project string, target *string) *projectCommit {
	bp := &projectCommit{map[string]interface{}{
		"organization": organization,
		"project":      project,
		"commitID":     commitID,
		"target":       ptr.From(target, ""),
	}}

	return bp
}

type projectCommit struct {
	vars map[string]interface{}
}

func (b *projectCommit) Query() string {
	return `
query ($commitID: String!, $organization: String!, $project: String!, $target: String) {
  project(organization: $organization, project: $project) {
    ... on Project {
      commit(vcsRef: $commitID) {
        ... on Commit {
          __typename
          expr
          commitId
          parentId
          atTime
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
                  ingredientID
                  ingredientVersionID
                  revision
                  name
                  namespace
                  version
                  licenses
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
        ... on Error {
          __typename
          message
        }
      }
    }
    ... on Error {
      __typename
      message
    }
  }
}
`
}

func (b *projectCommit) Vars() (map[string]interface{}, error) {
	return b.vars, nil
}
