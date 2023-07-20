package request

import "github.com/ActiveState/cli/internal/logging"

func BuildPlanByCommitID(commitID string) *buildPlanByCommitID {
	logging.Debug("BuildPlanByCommitID")
	bp := &buildPlanByCommitID{map[string]interface{}{
		"commitID": commitID,
	}}

	return bp
}

type buildPlanByCommitID struct {
	vars map[string]interface{}
}

func (b *buildPlanByCommitID) Query() string {
	return `
query ($commitID: ID!) {
  commit(commitId: $commitID) {
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
            ... on ArtifactFailed {
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
        ... on Error {
          message
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
`
}

func (b *buildPlanByCommitID) Vars() map[string]interface{} {
	return b.vars
}
