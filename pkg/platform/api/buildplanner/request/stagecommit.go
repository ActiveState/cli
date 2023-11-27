package request

import "github.com/ActiveState/cli/pkg/platform/runtime/buildexpression"

func StageCommit(owner, project, parentCommit, description string, expression *buildexpression.BuildExpression) *buildPlanByStageCommit {
	return &buildPlanByStageCommit{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"parentCommit": parentCommit,
		"description":  description,
		"expr":         expression,
	}}
}

type buildPlanByStageCommit struct {
	vars map[string]interface{}
}

func (b *buildPlanByStageCommit) Query() string {
	return `
mutation ($organization: String!, $project: String!, $parentCommit: ID, $description: String!, $expr:BuildExpr!) {
  stageCommit(input:{organization:$organization, project:$project, parentCommitId:$parentCommit, description:$description, expr:$expr}) {
    ... on Commit {
      __typename
			expr
      commitId
      build {
        __typename
        ... on BuildStarted {
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
                jsonPath
                error
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
    ... on ParseError {
      __typename
      message
      path
    }
    ... on NotFound {
      __typename
      message
      type
      resource
      mayNeedAuthentication
    }
    ... on Error {
      __typename
      message
    }
    ... on NoChangeSinceLastCommit {
      __typename
      commitId
      message
    }
    ... on Forbidden {
      __typename
      operation
      message
      resource
    }
    ... on HeadOnBranchMoved {
      __typename
      commitId
      branchId
      message
    }
  }
}
`
}

func (b *buildPlanByStageCommit) Vars() map[string]interface{} {
	return b.vars
}
