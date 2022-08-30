package model

type BuildPlanStatus string

type ArtifactStatus string

const (
	// BuildPlan statuses
	BuildPlanning BuildPlanStatus = "PLANNING"
	BuildPlanned  BuildPlanStatus = "PLANNED"
	BuildBuilding BuildPlanStatus = "BUILDING"
	BuildReady    BuildPlanStatus = "READY"
	// Currently the POC does not have a failed status
	BuildFailed BuildPlanStatus = "FAILED"

	// Artifact statuses
	ArtifactNotSubmitted      ArtifactStatus = "NOT_SUBMITTED"
	ArtifactBlocked           ArtifactStatus = "BLOCKED"
	ArtifactFailedPermanently ArtifactStatus = "FAILED_PERMANENTLY"
	ArtifactFailedTransiently ArtifactStatus = "FAILED_TRANSIENTLY"
	ArtifactReady             ArtifactStatus = "READY"
	ArtifactRunning           ArtifactStatus = "RUNNING"
	ArtifactSkipped           ArtifactStatus = "SKIPPED"
	ArtifactSucceeded         ArtifactStatus = "SUCCEEDED"

	// BuildResultTypes
	BuildResultPlanning      = "BuildPlanning"
	BuildResultPlanned       = "BuildPlanned"
	BuildResultStarted       = "BuildStarted"
	BuildResultReady         = "BuildReady"
	BuildResultPlanningError = "BuildPlanningError"

	ProjectNotFoundType = "ProjectNotFound"
	CommitNotFoundType  = "CommitNotFound"

	// Target types
	TargetTypeSource = "Source"
	TargetTypeStep   = "Step"
)

type BuildPlan struct {
	Project Project `json:"project"`
}

type Project struct {
	Type   string `json:"__typename"`
	Commit Commit `json:"commit"`
	ProjectNotFound
}

type ProjectNotFound struct {
	Message string `json:"message"`
}

type Commit struct {
	Type  string `json:"__typename"`
	Build Build  `json:"build"`
	CommitNotFound
}

type CommitNotFound struct {
	Message string `json:"message"`
}

type Build struct {
	Type        string          `json:"__typename"`
	BuildPlanID string          `json:"buildPlanID"`
	Status      BuildPlanStatus `json:"status"`
	Terminals   []NamedTarget   `json:"terminals"`
	Targets     []Target        `json:"targets"`

	// Error fields
	Error     string     `json:"error"`
	SubErrors []SubError `json:"subErrors"`
}

type NamedTarget struct {
	Tag       string   `json:"tag"`
	TargetIDs []string `json:"targetIDs"`
}

type Target struct {
	Type                string   `json:"__typename"`
	TargetID            string   `json:"targetID"`
	MimeType            string   `json:"mimeType"`
	GeneratedBy         string   `json:"generatedBy"`
	RuntimeDependencies []string `json:"runtimeDependencies"`
	Status              string   `json:"status"`
	URL                 string   `json:"url"`
	LogURL              string   `json:"logURL"`
	Checksum            string   `json:"checksum"`
	Errors              []string `json:"errors"`
	Attempts            string   `json:"attempts"`
	NextAttempt         string   `json:"nextAttempt"`

	// Step fields
	Inputs  []NamedTarget `json:"inputs"`
	Outputs []string      `json:"outputs"`

	// Source fields
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
}

type SubError struct {
	Type             string   `json:"__typename"`
	Path             string   `json:"path"`
	Message          string   `json:"message"`
	IsTransient      bool     `json:"isTransient"`
	ValidationErrors []string `json:"validationErrors"`
	RemediableSolveError
}

type RemediableSolveError struct {
	SuggestedRemediation
}

type SuggestedRemediation struct {
	RemediationType string   `json:"remediationType"`
	Command         string   `json:"command"`
	Parameters      []string `json:"parameters"`
}
