package request

import (
	model "github.com/ActiveState/cli/pkg/platform/api/buildplanner"
)

func PushCommit(owner, project, parentCommit, branchRef, description string, script model.BuildExpression) *buildPlanByPushCommit {
	return &buildPlanByPushCommit{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"parentCommit": parentCommit,
		"branchRef":    branchRef,
		"description":  description,
		"script":       script,
	}}
}

type buildPlanByPushCommit struct {
	vars map[string]interface{}
}

func (b *buildPlanByPushCommit) Query() string {
	return `
mutation ($organization: String!, $project: String!, $parentCommit: String!, $branchRef: String!, $script:BuildScript! $description: String!) {
  pushCommit(input:{org:$organization, project:$project, parentCommit:$parentCommit, script:$script, branchRef:$branchRef, description:$description}) {
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

func (b *buildPlanByPushCommit) Vars() map[string]interface{} {
	return b.vars
}
