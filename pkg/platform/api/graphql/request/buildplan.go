package request

func BuildPlanByCommitID(commitID string) *buildPlanByCommitID {
	return &buildPlanByCommitID{map[string]interface{}{
		"commitID": commitID,
	}}
}

type buildPlanByCommitID struct {
	vars map[string]interface{}
}

// TODO: Add error handling to this query
func (b *buildPlanByCommitID) Query() string {
	return `query($commitID: ID!) {
		execute(commitID: $commitID) {
			... on BuildPlan {
			buildPlanID
			status
			terminals {
				tag
				targetIDs
			}
			targets {
				... on Source {
					__typename
					targetID
					namespace
					name
					version
					revision
					ingredientID
					ingredientVersionID
				}
				... on Step {
					__typename
					targetID
					name
					inputs {
						__typename
						tag
						targetIDs
					}
					outputs
				}
				... on ArtifactSucceeded {
					__typename
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
					runningSince
				}
				... on ArtifactTransientlyFailed {
					__typename
					targetID
					mimeType
					generatedBy
					runtimeDependencies
					status
					lastBuildTimestamp
					buildTimeMs
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
					lastBuildTimestamp
					buildTimeMs
					logURL
					errors
				}
			}
		}
	}
}`
}

func (b *buildPlanByCommitID) Vars() map[string]interface{} {
	return b.vars
}
