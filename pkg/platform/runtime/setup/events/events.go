package events

import (
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
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

type Event struct{}

type Eventer interface {
	IsEvent() Event
}

type Start struct {
	RecipeID strfmt.UUID

	RequiresBuild bool
	ArtifactNames artifact.Named
	LogFilePath   string

	ArtifactsToBuild    []artifact.ArtifactID
	ArtifactsToDownload []artifact.ArtifactID
	ArtifactsToInstall  []artifact.ArtifactID
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
}

func (BuildFailure) IsEvent() Event {
	return Event{}
}

type ArtifactBuildStarted struct {
	ArtifactID artifact.ArtifactID
	FromCache  bool
}

func (ArtifactBuildStarted) IsEvent() Event {
	return Event{}
}

type ArtifactBuildProgress struct {
	ArtifactID   artifact.ArtifactID
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
	ArtifactID   artifact.ArtifactID
	LogURI       string
	ErrorMessage string
}

func (ArtifactBuildFailure) IsEvent() Event {
	return Event{}
}

type ArtifactBuildSuccess struct {
	ArtifactID artifact.ArtifactID
	LogURI     string
}

func (ArtifactBuildSuccess) IsEvent() Event {
	return Event{}
}

type ArtifactDownloadStarted struct {
	ArtifactID artifact.ArtifactID
	TotalSize  int
}

func (ArtifactDownloadStarted) IsEvent() Event {
	return Event{}
}

type ArtifactDownloadSkipped struct {
	ArtifactID artifact.ArtifactID
}

func (ArtifactDownloadSkipped) IsEvent() Event {
	return Event{}
}

type ArtifactDownloadProgress struct {
	ArtifactID      artifact.ArtifactID
	IncrementBySize int
}

func (ArtifactDownloadProgress) IsEvent() Event {
	return Event{}
}

type ArtifactDownloadFailure struct {
	ArtifactID artifact.ArtifactID
	Error      error
}

func (ArtifactDownloadFailure) IsEvent() Event {
	return Event{}
}

type ArtifactDownloadSuccess struct {
	ArtifactID artifact.ArtifactID
}

func (ArtifactDownloadSuccess) IsEvent() Event {
	return Event{}
}

type ArtifactInstallStarted struct {
	ArtifactID artifact.ArtifactID
	TotalSize  int
}

func (ArtifactInstallStarted) IsEvent() Event {
	return Event{}
}

type ArtifactInstallProgress struct {
	ArtifactID      artifact.ArtifactID
	IncrementBySize int
}

func (ArtifactInstallSkipped) IsEvent() Event {
	return Event{}
}

type ArtifactInstallSkipped struct {
	ArtifactID artifact.ArtifactID
}

func (ArtifactInstallProgress) IsEvent() Event {
	return Event{}
}

type ArtifactInstallFailure struct {
	ArtifactID artifact.ArtifactID
	Error      error
}

func (ArtifactInstallFailure) IsEvent() Event {
	return Event{}
}

type ArtifactInstallSuccess struct {
	ArtifactID artifact.ArtifactID
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
