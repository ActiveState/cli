package model

type BuildPlanStatusEnum string

type ArtifactStatus string

const (
	Planning BuildPlanStatusEnum = "PLANNING"
	Planned  BuildPlanStatusEnum = "PLANNED"
	Building BuildPlanStatusEnum = "BUILDING"
	Ready    BuildPlanStatusEnum = "READY"
	// TODO: Currently the POC does not have a failed status
	Failed BuildPlanStatusEnum = "FAILED"
)

const (
	ArtifactNotSubmitted      ArtifactStatus = "NOT_SUBMITTED"
	ArtifactBlocked           ArtifactStatus = "BLOCKED"
	ArtifactFailedPermanently ArtifactStatus = "FAILED_PERMANENTLY"
	ArtifactFailedTransiently ArtifactStatus = "FAILED_TRANSIENTLY"
	ArtifactReady             ArtifactStatus = "READY"
	ArtifactRunning           ArtifactStatus = "RUNNING"
	ArtifactSkipped           ArtifactStatus = "SKIPPED"
	ArtifactSucceeded         ArtifactStatus = "SUCCEEDED"
)

type BuildPlan struct {
	BPProject BPProject `json:"project"`
}

type BPProject struct {
	Commit BPCommit `json:"commit"`
}

type BPCommit struct {
	Build Build `json:"build"`
}

type Build struct {
	Terminals []Terminals `json:"terminals"`
	Status    string      `json:"status"`
	Targets   []Target    `json:"targets"`
	Error     string      `json:"error"`
	// TODO: Temporary workaround, remove after dependency resolution functions are updated
	Steps   []Step
	Sources []Source
}

type Terminals struct {
	Tag       string   `json:"tag"`
	TargetIDs []string `json:"targetIDs"`
}

type Target struct {
	TypeName            string   `json:"__typename"`
	TargetID            string   `json:"targetID"`
	Name                string   `json:"name"`
	Namespace           string   `json:"namespace"`
	Version             string   `json:"version"`
	MimeType            string   `json:"mimeType"`
	GeneratedBy         string   `json:"generatedBy"`
	Status              string   `json:"status"`
	URL                 string   `json:"url"`
	LogURL              string   `json:"logURL"`
	Checksum            string   `json:"checksum"`
	Image               string   `json:"image"`
	Command             string   `json:"command"`
	Inputs              []Input  `json:"inputs"`
	Outputs             []string `json:"outputs"`
	RuntimeDependencies []string `json:"runtimeDependencies"`
	Errors              []string `json:"errors"`
}

type Step struct {
	TargetID string   `json:"targetID"`
	Name     string   `json:"name"`
	Image    string   `json:"image"`
	Command  string   `json:"command"`
	Inputs   []Input  `json:"inputs"`
	Outputs  []string `json:"outputs"`
}

type Source struct {
	TargetID  string `json:"targetID"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Version   string `json:"version"`
}

type Input struct {
	Tag       string   `json:"tag"`
	TargetIDs []string `json:"targetIDs"`
}
