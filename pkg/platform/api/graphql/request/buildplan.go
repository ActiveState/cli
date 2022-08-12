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
	return `query ($commitID: ID!) {
		execute(commitID: $commitID) {
			... on BuildPlan {
			status
			terminals {
				tag
				targetIDs
			}
			resolvedRequirements {
				resolvedSource
				requirement {
				name
				namespace
				versionRequirements {
					Comparator
					Version
				}
				}
			}
			sources {
				targetID
				name
				namespace
				version
			}
			artifacts {
				... on ArtifactSucceeded {
				targetID
				mimeType
				generatedBy
				status
				url
				logURL
				checksum
				runtimeDependencies
				}
				... on ArtifactUnbuilt {
				targetID
				}
				... on ArtifactBuilding {
				targetID
				}
				... on ArtifactTransientlyFailed {
				targetID
				errors
				}
				... on ArtifactPermanentlyFailed {
				targetID
				errors
				}
			}
			steps {
				targetID
				name
				inputs {
				tag
				targetIDs
				}
				outputs
			}
			}
		}
	}`
}

func (b *buildPlanByCommitID) Vars() map[string]interface{} {
	return b.vars
}
