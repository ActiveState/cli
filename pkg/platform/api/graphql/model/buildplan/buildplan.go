package model

const (
	// BuildPlan statuses
	BuildPlanning = "PLANNING"
	BuildPlanned  = "PLANNED"
	BuildBuilding = "BUILDING"
	BuildReady    = "READY"
	// Currently the POC does not have a failed status
	BuildFailed = "FAILED"

	// Artifact statuses
	ArtifactNotSubmitted      = "NOT_SUBMITTED"
	ArtifactBlocked           = "BLOCKED"
	ArtifactFailedPermanently = "FAILED_PERMANENTLY"
	ArtifactFailedTransiently = "FAILED_TRANSIENTLY"
	ArtifactReady             = "READY"
	ArtifactRunning           = "RUNNING"
	ArtifactSkipped           = "SKIPPED"
	ArtifactSucceeded         = "SUCCEEDED"

	// BuildResultTypes
	BuildResultPlanning      = "BuildPlanning"
	BuildResultPlanned       = "BuildPlanned"
	BuildResultStarted       = "BuildStarted"
	BuildResultReady         = "BuildReady"
	BuildResultPlanningError = "BuildPlanningError"

	ProjectNotFoundType = "ProjectNotFound"
	CommitNotFoundType  = "CommitNotFound"

	// Tag types
	TagSource     = "src"
	TagDependency = "dep"
	TagBuilder    = "builder"
	TagOrphan     = "orphans"
)

type BuildPlan struct {
	Project Project `json:"project"`
}

type Project struct {
	Type   string `json:"__typename"`
	Commit Commit `json:"commit"`

	// Error fields
	Message string `json:"message"`
}

type Commit struct {
	Type  string `json:"__typename"`
	Build Build  `json:"build"`

	// Error fields
	Message string `json:"message"`
}

type Build struct {
	Type        string        `json:"__typename"`
	BuildPlanID string        `json:"buildPlanID"`
	Status      string        `json:"status"`
	Terminals   []NamedTarget `json:"terminals"`
	Artifacts   []Artifact    `json:"artifacts"`
	Steps       []Step        `json:"steps"`
	Sources     []Source      `json:"sources"`

	// Error fields
	Error     string     `json:"error"`
	SubErrors []SubError `json:"subErrors"`
}

type NamedTarget struct {
	Tag       string   `json:"tag"`
	TargetIDs []string `json:"targetIDs"`
}

type Artifact struct {
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
}

type Step struct {
	TargetID string        `json:"targetID"`
	Inputs   []NamedTarget `json:"inputs"`
	Outputs  []string      `json:"outputs"`
}

type Source struct {
	TargetID  string `json:"targetID"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
}

type PlanningError struct {
	Error    string `json:"error"`
	SubError SubError
}

type SubError struct {
	Type                  string   `json:"__typename"`
	Path                  string   `json:"path"`
	Message               string   `json:"message"`
	IsTransient           bool     `json:"isTransient"`
	ValidationErrors      []string `json:"validationErrors"`
	RemediableSolverError RemediableSolveError
}

type RemediableSolveError struct {
	Path                 string `json:"path"`
	Message              string `json:"message"`
	IsTransient          bool   `json:"isTransient"`
	ErrorType            string `json:"errorType"`
	SuggestedRemediation SolverErrorRemediation
}

type SolverErrorRemediation struct {
	RemediationType string   `json:"remediationType"`
	Command         string   `json:"command"`
	Parameters      []string `json:"parameters"`
}

type SolveErrorIncompatibility struct {
	Type                              string `json:"type"`
	SolveErrorPackageIncompatibility  SolveErrorPackageIncompatibility
	SolveErrorPlatformIncompatibility SolveErrorPlatformIncompatibility
}

type SolveErrorPackageIncompatibility struct {
	Feature   string `json:"feature"`
	Namespace string `json:"namespace"`
}

type SolveErrorPlatformIncompatibility struct {
	PlatformID     string `json:"platformID"`
	PlatformKernel string `json:"platformKernel"`
}
