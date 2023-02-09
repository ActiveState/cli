package model

const (
	// BuildPlan statuses
	Planning = "PLANNING"
	Planned  = "PLANNED"
	Building = "BUILDING"
	Ready    = "READY"

	// Artifact statuses
	ArtifactNotSubmitted      = "NOT_SUBMITTED"
	ArtifactBlocked           = "BLOCKED"
	ArtifactFailedPermanently = "FAILED_PERMANENTLY"
	ArtifactFailedTransiently = "FAILED_TRANSIENTLY"
	ArtifactReady             = "READY"
	ArtifactRunning           = "RUNNING"
	ArtifactSkipped           = "SKIPPED"
	ArtifactSucceeded         = "SUCCEEDED"

	// Types
	BuildResultPlanningError = "PlanningError"
	ProjectResultNotFound    = "ProjectNotFound"
	CommitResultNotFound     = "CommitNotFound"

	// Tag types
	TagSource     = "src"
	TagDependency = "dep"
	TagBuilder    = "builder"
	TagOrphan     = "orphans"
)

type BuildPlan struct {
	Project *Project `json:"project"`
}

type Project struct {
	Type   string  `json:"__typename"`
	Commit *Commit `json:"commit"`

	// Error field
	Message string `json:"message"`
}

type Commit struct {
	Type     string       `json:"__typename"`
	Graph    *BuildScript `json:"graph"`
	CommitID string       `json:"commitId"`
	Build    *Build       `json:"build"`

	// Error field
	Message string `json:"message"`
}

type Build struct {
	Type        string         `json:"__typename"`
	BuildPlanID string         `json:"buildPlanID"`
	Status      string         `json:"status"`
	Terminals   []*NamedTarget `json:"terminals"`
	Artifacts   []*Artifact    `json:"artifacts"`
	Steps       []*Step        `json:"steps"`
	Sources     []*Source      `json:"sources"`
	*PlanningError
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

	// Error fields
	Errors      []string `json:"errors"`
	Attempts    string   `json:"attempts"`
	NextAttempt string   `json:"nextAttempt"`
}

type Step struct {
	TargetID string         `json:"targetID"`
	Inputs   []*NamedTarget `json:"inputs"`
	Outputs  []string       `json:"outputs"`
}

type Source struct {
	TargetID  string `json:"targetID"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
}

type PlanningError struct {
	Error     string      `json:"error"`
	SubErrors []*SubError `json:"subErrors"`
}

type SubError struct {
	Type             string                        `json:"__typename"`
	Path             string                        `json:"path"`
	Message          string                        `json:"message"`
	IsTransient      bool                          `json:"isTransient"`
	ValidationErrors []*SolverErrorValidationError `json:"validationErrors"`
	Remediations     []*SolverErrorRemediation     `json:"suggestedRemediations"`
	*RemediableSolveError
}

type SolverErrorValidationError struct {
	JSONPath string `json:"jsonPath"`
	Error    string `json:"error"`
}

type RemediableSolveError struct {
	ErrorType string `json:"errorType"`
}

type SolverErrorRemediation struct {
	RemediationType string `json:"remediationType"`
	Command         string `json:"command"`
}

type SolveErrorIncompatibility struct {
	Type string `json:"type"`
	*SolveErrorPackageIncompatibility
	*SolveErrorPlatformIncompatibility
}

type SolveErrorPackageIncompatibility struct {
	Feature   string `json:"feature"`
	Namespace string `json:"namespace"`
}

type SolveErrorPlatformIncompatibility struct {
	PlatformID     string `json:"platformID"`
	PlatformKernel string `json:"platformKernel"`
}
