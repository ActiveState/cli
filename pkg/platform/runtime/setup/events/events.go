package events

import (
	"github.com/go-openapi/strfmt"
)

/*
These events are intentionally low level and with minimal abstraction so as to avoid this mechanic becoming unwieldy.
The naming format of events should be in the form of <Component>[<Action>]<Outcome>.
*/

type Handler interface {
	Handle(ev Eventer) error
	Close() error
}

type VoidHandler struct {
}

func (v *VoidHandler) Handle(Eventer) error {
	return nil
}

func (v *VoidHandler) Close() error {
	return nil
}

type Event struct{}

type Eventer interface {
	IsEvent() Event
}

type Start struct {
	RecipeID strfmt.UUID

	RequiresBuild bool
	Artifacts     map[strfmt.UUID]string
	LogFilePath   string

	ArtifactsToBuild    []strfmt.UUID
	ArtifactsToDownload []strfmt.UUID
	ArtifactsToInstall  []strfmt.UUID
}

func (Start) IsEvent() Event {
	return Event{}
}

type Success struct {
}

func (Success) IsEvent() Event {
	return Event{}
}

type Failure struct {
}

func (Failure) IsEvent() Event {
	return Event{}
}

type BuildSkipped struct {
}

func (BuildSkipped) IsEvent() Event {
	return Event{}
}

type BuildStarted struct {
	LogFilePath string
}

func (BuildStarted) IsEvent() Event {
	return Event{}
}

type BuildSuccess struct {
}

func (BuildSuccess) IsEvent() Event {
	return Event{}
}

type BuildFailure struct {
	Message string
}

func (BuildFailure) IsEvent() Event {
	return Event{}
}

type ArtifactBuildStarted struct {
	ArtifactID strfmt.UUID
	FromCache  bool
}

func (ArtifactBuildStarted) IsEvent() Event {
	return Event{}
}

type ArtifactBuildProgress struct {
	ArtifactID   strfmt.UUID
	LogTimestamp string
	LogLevel     string // eg. (INFO/ERROR/...)
	LogChannel   string // channel through which this log line was generated (stdout/stderr/...)
	LogMessage   string
	LogSource    string // source of this log (eg., builder/build-wrapper/...)
}

func (ArtifactBuildProgress) IsEvent() Event {
	return Event{}
}

type ArtifactBuildFailure struct {
	ArtifactID   strfmt.UUID
	LogURI       string
	ErrorMessage string
}

func (ArtifactBuildFailure) IsEvent() Event {
	return Event{}
}

type ArtifactBuildSuccess struct {
	ArtifactID strfmt.UUID
	LogURI     string
}

func (ArtifactBuildSuccess) IsEvent() Event {
	return Event{}
}

type ArtifactDownloadStarted struct {
	ArtifactID strfmt.UUID
	TotalSize  int
}

func (ArtifactDownloadStarted) IsEvent() Event {
	return Event{}
}

type ArtifactDownloadSkipped struct {
	ArtifactID strfmt.UUID
}

func (ArtifactDownloadSkipped) IsEvent() Event {
	return Event{}
}

type ArtifactDownloadProgress struct {
	ArtifactID      strfmt.UUID
	IncrementBySize int
}

func (ArtifactDownloadProgress) IsEvent() Event {
	return Event{}
}

type ArtifactDownloadFailure struct {
	ArtifactID strfmt.UUID
	Error      error
}

func (ArtifactDownloadFailure) IsEvent() Event {
	return Event{}
}

type ArtifactDownloadSuccess struct {
	ArtifactID strfmt.UUID
}

func (ArtifactDownloadSuccess) IsEvent() Event {
	return Event{}
}

type ArtifactInstallStarted struct {
	ArtifactID strfmt.UUID
	TotalSize  int
}

func (ArtifactInstallStarted) IsEvent() Event {
	return Event{}
}

type ArtifactInstallProgress struct {
	ArtifactID      strfmt.UUID
	IncrementBySize int
}

func (ArtifactInstallSkipped) IsEvent() Event {
	return Event{}
}

type ArtifactInstallSkipped struct {
	ArtifactID strfmt.UUID
}

func (ArtifactInstallProgress) IsEvent() Event {
	return Event{}
}

type ArtifactInstallFailure struct {
	ArtifactID strfmt.UUID
	Error      error
}

func (ArtifactInstallFailure) IsEvent() Event {
	return Event{}
}

type ArtifactInstallSuccess struct {
	ArtifactID strfmt.UUID
}

func (ArtifactInstallSuccess) IsEvent() Event {
	return Event{}
}

type SolveStart struct{}

func (SolveStart) IsEvent() Event {
	return Event{}
}

type SolveError struct {
	Error error
}

func (SolveError) IsEvent() Event {
	return Event{}
}

type SolveSuccess struct{}

func (SolveSuccess) IsEvent() Event {
	return Event{}
}
