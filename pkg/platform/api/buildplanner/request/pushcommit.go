package request

import (
	model "github.com/ActiveState/cli/pkg/platform/api/buildplanner/model"
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
