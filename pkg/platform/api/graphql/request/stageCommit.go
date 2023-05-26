package request

import (
	model "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
)

func StageCommit(owner, project, parentCommit string, script model.BuildExpression) *buildPlanByStageCommit {
	return &buildPlanByStageCommit{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"parentCommit": parentCommit,
		"script":       script,
	}}
}

type buildPlanByStageCommit struct {
	vars map[string]interface{}
}

func (b *buildPlanByStageCommit) Query() string {
	return `
mutation ($organization: String!, $project: String!, $parentCommit: String!, $script:BuildScript!) {
  stageCommit(input:{org:$organization, project:$project, parentCommit:$parentCommit, script:$script}) {
    ... on Commit {
      __typename
			script
      commitId
      build {
        __typename
        ... on BuildReady {
          buildLogIds {
            id
            type
          }
        }
        ... on BuildStarted {
          buildLogIds {
            id
            type
          }
        }
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
            ... on ArtifactSucceeded {
              __typename
              targetID
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
              targetID
              mimeType
              generatedBy
              runtimeDependencies
              status
            }
            ... on ArtifactBuilding {
              __typename
              targetID
              mimeType
              generatedBy
              runtimeDependencies
              status
            }
            ... on ArtifactTransientlyFailed {
              __typename
              targetID
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
              targetID
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
      message
    }
    ... on Error{
      message
    }
  }
}
`
}

func (b *buildPlanByStageCommit) Vars() map[string]interface{} {
	return b.vars
}
