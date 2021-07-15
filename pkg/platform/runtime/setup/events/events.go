// Package events handles the processing of setup events. As these events can be
// sent in parallel, all events are collected in a single go-routine to simplify
// their processing.
//
// The events are generated in the RuntimeEventProducer which exposes an events
// channel that can be consumed by the RuntimeEventConsumer. The consumer then
// delegates the event handling to digesters: ProgressDigester and
// ChangeSummaryDigester
//                                     +--- RuntimeEventHandler -------------------------------------+
//                                     |                                                             |
//                                     |                                   +-----------------------+ |
//                                     |                               ,-> | ChangeSummaryDigester | |
// +----------------------+            |  +----------------------+    /    +-----------------------+ |
// | RuntimeEventProducer | ---------> |  | RuntimeEventConsumer | ---+                              |
// +----------------------+  .Events() |  +----------------------+    \    +------------------+      |
//                                     |                               `-> | ProgressDigester |      |
//                                     |                                   +------------------+      |
//                                     +-------------------------------------------------------------+
// The runbits package has default implementations for digesters, and the
// RuntimeEventHandler combines the consumer with its digesters.
package events

// This file contains the definition of all events that the RuntimeEventProducer creates.

import (
	"fmt"
	"time"

	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

// SetupStep is the step the runtime setup routine is currently at.
type SetupStep int

const (
	// Build refers to a remote building artifact
	Build SetupStep = iota
	// Download refers to a currently downloading artifact
	Download
	// Unpack refers to the step where an artifact tarball is currently being unpacked
	Unpack
	// Install refers to all the post-processing that needs to happen to get an artifact ready for use.
	Install
)

func (s SetupStep) String() string {
	switch s {
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

// SetupEventer is the interface that every setup event should implement
type SetupEventer interface {
	// String returns a description of the event
	String() string
}

// ArtifactSetupEventer describes the methods for an event that reports progress on a specific artifact
type ArtifactSetupEventer interface {
	SetupEventer
	Step() SetupStep
	ArtifactID() artifact.ArtifactID
}

// baseEvent is a re-usable struct that implements the Step() and String() methods
type baseEvent struct {
	name string
	step SetupStep
}

func newBaseEvent(name string, step SetupStep) baseEvent {
	return baseEvent{name, step}
}

func (be baseEvent) String() string {
	return be.name
}

func (be baseEvent) Step() SetupStep {
	return be.step
}

// artifactBaseEvent is a re-usable struct that implements the ArtifactEventer interface
type artifactBaseEvent struct {
	baseEvent
	artifactID artifact.ArtifactID
}

func newArtifactBaseEvent(suffix string, step SetupStep, artifactID artifact.ArtifactID) artifactBaseEvent {
	return artifactBaseEvent{newBaseEvent(fmt.Sprintf("artifact_%s_%s", step.String(), suffix), step), artifactID}
}

type artifactBuildEvent struct {
	artifactBaseEvent
	logURI string
}

func newArtifactBuildEvent(suffix string, step SetupStep, artifactID artifact.ArtifactID, logURI string) artifactBuildEvent {
	return artifactBuildEvent{newArtifactBaseEvent(suffix, step, artifactID), logURI}
}

// ArtifactResolverEvents forwards a function to resolve artifact names (as soon as that information becomes available)
type ArtifactResolverEvent struct {
	resolver     ArtifactResolver
	downloadable []artifact.ArtifactDownload
}

// Resolver returns a function that resolves artifact names
func (ae ArtifactResolverEvent) Resolver() ArtifactResolver {
	return ae.resolver
}

// Resolver returns a function that resolves artifact names
func (ae ArtifactResolverEvent) DownloadableArtifacts() []artifact.ArtifactDownload {
	return ae.downloadable
}

func (ae ArtifactResolverEvent) String() string {
	return "artifact_resolver"
}

func newArtifactResolverEvent(resolver ArtifactResolver, downloadable []artifact.ArtifactDownload) ArtifactResolverEvent {
	return ArtifactResolverEvent{resolver, downloadable}
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
	totalBuilds int
}

func (be BuildStartEvent) Total() int {
	return be.totalBuilds
}

func newBuildStartEvent(totalBuilds int) BuildStartEvent {
	return BuildStartEvent{newBaseEvent("build_start", Build), totalBuilds}
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
	total int
}

func newArtifactStartEvent(step SetupStep, artifactID artifact.ArtifactID, total int) ArtifactStartEvent {
	return ArtifactStartEvent{newArtifactBaseEvent("start", step, artifactID), total}
}

// Total returns the total number of elements (usually bytes) that we expect for this artifact in the given step
func (ase ArtifactStartEvent) Total() int {
	return ase.total
}

type ArtifactBuildProgressEvent struct {
	artifactBaseEvent

	timeStamp string
	message   string
	facility  string
	pipeName  string
	source    string
}

func (ae ArtifactBuildProgressEvent) TimeStamp() string {
	return ae.timeStamp
}

func (ae ArtifactBuildProgressEvent) Message() string {
	return ae.message
}

func (ae ArtifactBuildProgressEvent) Facility() string {
	return ae.facility
}

func (ae ArtifactBuildProgressEvent) PipeName() string {
	return ae.pipeName
}

func (ae ArtifactBuildProgressEvent) Source() string {
	return ae.source
}

func newArtifactBuildProgressEvent(artifactID artifact.ArtifactID, timeStamp string, msg, facility, pipeName, source string) ArtifactBuildProgressEvent {
	return ArtifactBuildProgressEvent{newArtifactBaseEvent("progress", Build, artifactID), timeStamp, msg, facility, pipeName, source}
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

func newArtifactProgressEvent(step SetupStep, artifactID artifact.ArtifactID, increment int) ArtifactProgressEvent {
	return ArtifactProgressEvent{newArtifactBaseEvent("progress", step, artifactID), increment}
}

// ArtifactCompleteEvent is sent when an artifact step completed
type ArtifactCompleteEvent struct {
	artifactBuildEvent
}

func newArtifactCompleteEvent(step SetupStep, artifactID artifact.ArtifactID, logURI string) ArtifactCompleteEvent {
	return ArtifactCompleteEvent{newArtifactBuildEvent("complete", step, artifactID, logURI)}
}

// ArtifactFailureEvent is sent when an artifact failed to process through the given step
type ArtifactFailureEvent struct {
	artifactBuildEvent
	errorMessage string
}

// Failure returns a description of the error message
func (fe ArtifactFailureEvent) Failure() string {
	return fe.errorMessage
}

func newArtifactFailureEvent(step SetupStep, artifactID artifact.ArtifactID, logURI, errorMessage string) ArtifactFailureEvent {
	return ArtifactFailureEvent{newArtifactBuildEvent("failure", step, artifactID, logURI), errorMessage}
}

// ChangeSummaryEvent is sent when a the information to summarize the changes introduced by this runtime is available
type ChangeSummaryEvent struct {
	artifacts map[artifact.ArtifactID]artifact.ArtifactRecipe
	requested artifact.ArtifactChangeset
	changed   artifact.ArtifactChangeset
}

func (cse ChangeSummaryEvent) String() string {
	return "change_summary"
}

// Artifacts returns the map of ArtifactRecipe structs extracted from the recipe
func (cse ChangeSummaryEvent) Artifacts() map[artifact.ArtifactID]artifact.ArtifactRecipe {
	return cse.artifacts
}

// RequestedChangeset returns the changeset information for artifacts that the user requested to change (add/remove/update)
func (cse ChangeSummaryEvent) RequestedChangeset() artifact.ArtifactChangeset {
	return cse.requested
}

// CompleteChangeset returns the changeset information for all artifacts that have changed relative to the locally installed runtime
func (cse ChangeSummaryEvent) CompleteChangeset() artifact.ArtifactChangeset {
	return cse.changed
}

func newChangeSummaryEvent(artifacts map[artifact.ArtifactID]artifact.ArtifactRecipe, requested artifact.ArtifactChangeset, changed artifact.ArtifactChangeset) ChangeSummaryEvent {
	return ChangeSummaryEvent{
		artifacts, requested, changed,
	}
}

type HeartbeatEvent struct {
	baseEvent
	timeStamp time.Time
}

func (he HeartbeatEvent) TimeStamp() time.Time {
	return he.timeStamp
}

func newHeartbeatEvent(timeStamp time.Time) HeartbeatEvent {
	return HeartbeatEvent{newBaseEvent("heartbeat", Build), timeStamp}
}
