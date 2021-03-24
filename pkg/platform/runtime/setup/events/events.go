package events

import (
	"fmt"

	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

// ArtifactSetupStep is the step at which the artifact processing routine is currently at.
type ArtifactSetupStep int

const (
	// Build refers to a remote building artifact
	Build ArtifactSetupStep = iota
	// Download refers to a currently downloading artifact
	Download
	// Unpack refers to the step where an artifact tarball is currently being unpacked
	Unpack
	// Install refers to all the post-processing that needs to happen to get an artifact ready for use.
	Install
)

func (ass ArtifactSetupStep) String() string {
	switch ass {
	case Build:
		return "build"
	case Download:
		return "download"
	case Unpack:
		return "unpack"
	case Install:
		return "install"
	default:
		return "invalid"
	}
}

// BaseEventer is the interface that every setup event should implement
type BaseEventer interface {
	String() string
}

// ArtifactEventer describes the methods for an event that reports progress on a specific artifact
type ArtifactEventer interface {
	BaseEventer
	Step() ArtifactSetupStep
	ArtifactID() artifact.ArtifactID
}

// baseEvent is a re-usable struct that implements the Step() and String() methods
type baseEvent struct {
	name string
	step ArtifactSetupStep
}

func newBaseEvent(name string, step ArtifactSetupStep) baseEvent {
	return baseEvent{name, step}
}

func (be baseEvent) String() string {
	return be.name
}

func (be baseEvent) Step() ArtifactSetupStep {
	return be.step
}

// artifactBaseEvent is a re-usable struct that implements the ArtifactEventer interface
type artifactBaseEvent struct {
	baseEvent
	artifactID artifact.ArtifactID
}

func newArtifactBaseEvent(suffix string, step ArtifactSetupStep, artifactID artifact.ArtifactID) artifactBaseEvent {
	return artifactBaseEvent{newBaseEvent(fmt.Sprintf("artifact_%s_%s", step.String(), suffix), step), artifactID}
}

// TotalArtifactEvent reports the number of total artifacts as soon as they are known
type TotalArtifactEvent struct {
	total int
}

// Total returns the number of artifacts that we are dealing with
func (te TotalArtifactEvent) Total() int {
	return te.total
}

func (te TotalArtifactEvent) String() string {
	return "artifact_total"
}

func newTotalArtifactEvent(total int) TotalArtifactEvent {
	return TotalArtifactEvent{total}
}

// BuildStartEvent reports the beginning of the remote build process
type BuildStartEvent struct {
	baseEvent
}

func newBuildStartEvent() BuildStartEvent {
	return BuildStartEvent{newBaseEvent("build_start", Build)}
}

// BuildCompleteEvent reports the successful completion of a build
type BuildCompleteEvent struct {
	baseEvent
}

func newBuildCompleteEvent() BuildCompleteEvent {
	return BuildCompleteEvent{newBaseEvent("build_complete", Build)}
}

func (be artifactBaseEvent) ArtifactID() artifact.ArtifactID {
	return be.artifactID
}

// ArtifactStartEvent is sent when an artifact enters a new processing step
type ArtifactStartEvent struct {
	artifactBaseEvent
	artifactName string
	total        int
}

func newArtifactStartEvent(step ArtifactSetupStep, artifactID artifact.ArtifactID, artifactName string, total int) ArtifactStartEvent {
	return ArtifactStartEvent{newArtifactBaseEvent("start", step, artifactID), artifactName, total}
}

// ArtifactName returns the name of the artifact that entered the new step
func (ase ArtifactStartEvent) ArtifactName() string {
	return ase.artifactName
}

// Total returns the total number of elements (usually bytes) that we expect for this artifact in the given step
func (ase ArtifactStartEvent) Total() int {
	return ase.total
}

// ArtifactProgressEvent is sent when the artifact has progressed in the given step
type ArtifactProgressEvent struct {
	artifactBaseEvent
	increment int
}

// Progress returns the increment by which the artifact has progressed
func (ue ArtifactProgressEvent) Progress() int {
	return ue.increment
}

func newArtifactProgressEvent(step ArtifactSetupStep, artifactID artifact.ArtifactID, increment int) ArtifactProgressEvent {
	return ArtifactProgressEvent{newArtifactBaseEvent("progress", step, artifactID), increment}
}

// ArtifactCompleteEvent is sent when an artifact step completed
type ArtifactCompleteEvent struct {
	artifactBaseEvent
}

func newArtifactCompleteEvent(step ArtifactSetupStep, artifactID artifact.ArtifactID) ArtifactCompleteEvent {
	return ArtifactCompleteEvent{newArtifactBaseEvent("complete", step, artifactID)}
}

// ArtifactFailureEvent is sent when an artifact failed to process through the given step
type ArtifactFailureEvent struct {
	artifactBaseEvent
	errorMessage string
}

// Failure returns a description of the error message
func (fe ArtifactFailureEvent) Failure() string {
	return fe.errorMessage
}

func newArtifactFailureEvent(step ArtifactSetupStep, artifactID artifact.ArtifactID, errorMessage string) ArtifactFailureEvent {
	return ArtifactFailureEvent{newArtifactBaseEvent("failure", step, artifactID), errorMessage}
}

type ChangeSummaryEvent struct {
	artifacts map[artifact.ArtifactID]artifact.ArtifactRecipe
	requested artifact.ArtifactChangeset
	changed   artifact.ArtifactChangeset
}

func (cse ChangeSummaryEvent) String() string {
	return "change_summary"
}

func (cse ChangeSummaryEvent) Artifacts() map[artifact.ArtifactID]artifact.ArtifactRecipe {
	return cse.artifacts
}

func (cse ChangeSummaryEvent) RequestedChangeset() artifact.ArtifactChangeset {
	return cse.requested
}

func (cse ChangeSummaryEvent) CompleteChangeset() artifact.ArtifactChangeset {
	return cse.changed
}

func newChangeSummaryEvent(artifacts map[artifact.ArtifactID]artifact.ArtifactRecipe, requested artifact.ArtifactChangeset, changed artifact.ArtifactChangeset) ChangeSummaryEvent {
	return ChangeSummaryEvent{
		artifacts, requested, changed,
	}
}
