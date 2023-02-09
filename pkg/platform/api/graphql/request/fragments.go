package request

const buildResultFragment = `
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
`
