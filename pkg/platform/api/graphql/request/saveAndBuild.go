package request

import model "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplan"

func SaveAndBuild(owner, project, parentCommit, branchRef, description string, graph *model.BuildGraph) *buildPlanBySaveAndBuild {
	return &buildPlanBySaveAndBuild{map[string]interface{}{
		"organization": owner,
		"project":      project,
		"parentCommit": parentCommit,
		"branchRef":    branchRef,
		"description":  description,
		"graph":        graph,
	}}
}

type buildPlanBySaveAndBuild struct {
	vars map[string]interface{}
}

func (b *buildPlanBySaveAndBuild) Query() string {
	return `
mutation ($organization: String!, $project: String!, $parentCommit: String!, $graph: BuildGraph!, $branchRef: String!, $description:String!) {
  saveAndBuild(organization: $organization, project: $project, parentCommit: $parentCommit, graph: $graph, branchRef: $branchRef, description:$description) {
    ... on Commit {
      __typename
      graph
	  commitId
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
    ... on CommitNotFound {
      message
    }
    ... on BuildSubmissionError {
      message
    }
  }
}
`
}

func (b *buildPlanBySaveAndBuild) Vars() map[string]interface{} {
	return b.vars
}
