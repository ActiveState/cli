package request

func BuildPlan(commitID, owner, project string) *buildPlanByCommitID {
	bp := &buildPlanByCommitID{map[string]interface{}{
		"commitID": commitID,
	}}

	if owner != "" {
		bp.vars["organization"] = owner
	}

	if project != "" {
		bp.vars["project"] = project
	}

	return bp
}

type buildPlanByCommitID struct {
	vars map[string]interface{}
}

func (b *buildPlanByCommitID) Query() string {
	return `
query ($commitID: String!) {
  commit(commitId: $commitID) {
    ... on Commit {
      __typename
      expr
      build {
        __typename
        ... on BuildReady {
          buildLogIds {
            id
            type
            platformId
          }
        }
        ... on BuildStarted {
          buildLogIds {
            id
            type
            platformId
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
      __typename
      message
    }
  }
}
`
}

func (b *buildPlanByCommitID) Vars() map[string]interface{} {
	return b.vars
}
