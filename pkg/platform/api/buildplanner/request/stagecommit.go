package request

import (
	"time"

	"github.com/ActiveState/cli/internal/rtutils/ptr"
)

func StageCommit(owner, project, parentCommit, description string, atTime *time.Time, expression []byte) *buildPlanByStageCommit {
	var timestamp *string
	if atTime != nil {
		timestamp = ptr.To(atTime.Format(time.RFC3339))
	}
	return &buildPlanByStageCommit{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"parentCommit": parentCommit,
		"description":  description,
		"expr":         string(expression),
		"atTime":       timestamp, // default to the latest timestamp
	}}
}

type buildPlanByStageCommit struct {
	vars map[string]interface{}
}

func (b *buildPlanByStageCommit) Query() string {
	return `
mutation ($organization: String!, $project: String!, $parentCommit: ID!, $description: String!, $atTime: DateTime, $expr: BuildExpr!) {
  stageCommit(
    input: {organization: $organization, project: $project, parentCommitId: $parentCommit, description: $description, atTime: $atTime, expr: $expr}
  ) {
    ... on Commit {
      __typename
      expr
      commitId
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

func (b *buildPlanByStageCommit) Vars() (map[string]interface{}, error) {
	return b.vars, nil
}
