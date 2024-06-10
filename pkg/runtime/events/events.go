package events

import (
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/go-openapi/strfmt"
)

/*
These events are intentionally low level and with minimal abstraction so as to avoid this mechanic becoming unwieldy.
The naming format of events should be in the form of <Component>[<Action>]<Outcome>.
*/

type Handler interface {
	Handle(ev Event) error
	Close() error
}

type Event interface {
	IsEvent()
}

type HandlerFunc func(Event) error

type VoidHandler struct {
}

func (v *VoidHandler) Handle(Event) error {
	return nil
}

func (v *VoidHandler) Close() error {
	return nil
}

type Start struct {
	RecipeID strfmt.UUID

	RequiresBuild bool
	LogFilePath   string

	ArtifactsToBuild    buildplan.ArtifactIDMap
	ArtifactsToDownload buildplan.ArtifactIDMap
	ArtifactsToInstall  buildplan.ArtifactIDMap
}

func (Start) IsEvent() {}

type Success struct {
}

func (Success) IsEvent() {}

type Failure struct {
	Error error
}

func (Failure) IsEvent() {}

type BuildStarted struct {
	LogFilePath string
}

func (BuildStarted) IsEvent() {}

type BuildSuccess struct {
}

func (BuildSuccess) IsEvent() {}

type BuildFailure struct {
	Message string
}

func (BuildFailure) IsEvent() {}

type ArtifactBuildStarted struct {
	ArtifactID strfmt.UUID
	FromCache  bool
}

func (ArtifactBuildStarted) IsEvent() {}

type ArtifactBuildProgress struct {
	ArtifactID   strfmt.UUID
	LogTimestamp string
	LogLevel     string // eg. (INFO/ERROR/...)
	LogChannel   string // channel through which this log line was generated (stdout/stderr/...)
	LogMessage   string
	LogSource    string // source of this log (eg., builder/build-wrapper/...)
}

func (ArtifactBuildProgress) IsEvent() {}

type ArtifactBuildFailure struct {
	ArtifactID   strfmt.UUID
	LogURI       string
	ErrorMessage string
}

func (ArtifactBuildFailure) IsEvent() {}

type ArtifactBuildSuccess struct {
	ArtifactID strfmt.UUID
	LogURI     string
}

func (ArtifactBuildSuccess) IsEvent() {}

type ArtifactDownloadStarted struct {
	ArtifactID strfmt.UUID
	TotalSize  int
}

func (ArtifactDownloadStarted) IsEvent() {}

type ArtifactDownloadProgress struct {
	ArtifactID      strfmt.UUID
	IncrementBySize int
}

func (ArtifactDownloadProgress) IsEvent() {}

type ArtifactDownloadFailure struct {
	ArtifactID strfmt.UUID
	Error      error
}

func (ArtifactDownloadFailure) IsEvent() {}

type ArtifactDownloadSuccess struct {
	ArtifactID strfmt.UUID
}

func (ArtifactDownloadSuccess) IsEvent() {}

type ArtifactInstallStarted struct {
	ArtifactID strfmt.UUID
}

func (ArtifactInstallStarted) IsEvent() {}

type ArtifactInstallFailure struct {
	ArtifactID strfmt.UUID
	Error      error
}

func (ArtifactInstallFailure) IsEvent() {}

type ArtifactInstallSuccess struct {
	ArtifactID strfmt.UUID
}

func (ArtifactInstallSuccess) IsEvent() {}

type ArtifactUninstallStarted struct {
	ArtifactID strfmt.UUID
}

func (ArtifactUninstallStarted) IsEvent() {}

type ArtifactUninstallFailure struct {
	ArtifactID strfmt.UUID
	Error      error
}

func (ArtifactUninstallFailure) IsEvent() {}

type ArtifactUninstallSuccess struct {
	ArtifactID strfmt.UUID
}

func (ArtifactUninstallSuccess) IsEvent() {}

type ArtifactUnpackStarted struct {
	ArtifactID strfmt.UUID
	TotalSize  int
}

func (ArtifactUnpackStarted) IsEvent() {}

type ArtifactUnpackProgress struct {
	ArtifactID      strfmt.UUID
	IncrementBySize int
}

func (ArtifactUnpackProgress) IsEvent() {}

type ArtifactUnpackFailure struct {
	ArtifactID strfmt.UUID
	Error      error
}

func (ArtifactUnpackFailure) IsEvent() {}

type ArtifactUnpackSuccess struct {
	ArtifactID strfmt.UUID
}

func (ArtifactUnpackSuccess) IsEvent() {}

type PostProcessStarted struct {
}

func (PostProcessStarted) IsEvent() {}

type PostProcessSuccess struct {
}

func (PostProcessSuccess) IsEvent() {}

type PostProcessFailure struct {
	Error error
}

func (PostProcessFailure) IsEvent() {}
