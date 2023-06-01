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
	NotFound                 = "NotFound"
	BuildResultPlanningError = "PlanningError"

	// Tag types
	TagSource     = "src"
	TagDependency = "dep"
	TagBuilder    = "builder"
	TagOrphan     = "orphans"

	// BuildLogID types
	BuildLogRecipeID = "RECIPE_ID"
	BuildRequestID   = "BUILD_REQUEST_ID"
)

type BuildPlan struct {
	Project *Project `json:"project"`
}

type PushCommitResult struct {
	Commit *Commit `json:"pushCommit"`
	*NotFoundError
}

type StageCommitResult struct {
	Commit *Commit `json:"stageCommit"`
	*NotFoundError
}

type NotFoundError struct {
	Message string `json:"message"`
}

type Error struct {
	Message string `json:"message"`
}

type Project struct {
	Type   string  `json:"__typename"`
	Commit *Commit `json:"commit"`
	*NotFoundError
}

type Commit struct {
	Type     string `json:"__typename"`
	Script   string `json:"script"`
	CommitID string `json:"commitId"`
	Build    *Build `json:"build"`
	*NotFoundError
}

type Build struct {
	Type        string         `json:"__typename"`
	BuildPlanID string         `json:"buildPlanID"`
	Status      string         `json:"status"`
	Terminals   []*NamedTarget `json:"terminals"`
	Artifacts   []*Artifact    `json:"artifacts"`
	Steps       []*Step        `json:"steps"`
	Sources     []*Source      `json:"sources"`
	BuildLogIDs []*BuildLogID  `json:"buildLogIds"`
	*PlanningError
}

type BuildLogID struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	PlatformID string `json:"platformID"`
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
	Error     string                 `json:"error"`
	SubErrors []*BuildScriptLocation `json:"subErrors"`
}

type BuildScriptLocation struct {
	Type             string                        `json:"__typename"`
	Path             string                        `json:"path"`
	Message          string                        `json:"message"`
	IsTransient      bool                          `json:"isTransient"`
	ValidationErrors []*SolverErrorValidationError `json:"validationErrors"`
	*RemediableSolveError
}

type SolverErrorValidationError struct {
	JSONPath string `json:"jsonPath"`
	Error    string `json:"error"`
}

type RemediableSolveError struct {
	ErrorType         string                       `json:"errorType"`
	Remediations      []*SolverErrorRemediation    `json:"suggestedRemediations"`
	Requirements      []*Requirement               `json:"requirements"`
	Incompatibilities []*SolveErrorIncompatibility `json:"incompatibilities"`
}

type SolverErrorRemediation struct {
	RemediationType string `json:"solveErrorRemediationType"`
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
