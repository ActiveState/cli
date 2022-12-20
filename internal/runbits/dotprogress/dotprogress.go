package dotprogress

import (
	"time"

	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

const (
	// DefaultInterval is the default interval for the dot progress
	DefaultInterval = 5 * time.Second
)

type DotProgress struct {
	*output.DotProgress
}

func NewRuntimeProgress(out output.Outputer) *DotProgress {
	return &DotProgress{output.NewDotProgress(out, locale.T("runtime_dotprogress_start"), DefaultInterval)}
}

func (d *DotProgress) Close() error {
	d.Stop(locale.T("runtime_dotprogress_stop"))
	return nil
}

func (d *DotProgress) SolverError(serr *model.SolverError) error { return nil }

func (d *DotProgress) SolverStart() error { return nil }

func (d *DotProgress) SolverSuccess() error { return nil }

func (d *DotProgress) BuildStarted(totalArtifacts int64) error { return nil }

func (d *DotProgress) BuildCompleted(withFailures bool) error { return nil }

func (d *DotProgress) InstallationStarted(totalArtifacts int64) error { return nil }

func (d *DotProgress) InstallationStatusUpdate(current, total int64) error { return nil }

func (d *DotProgress) InstallationCompleted(withFailures bool) error { return nil }

func (d *DotProgress) BuildArtifactStarted(artifactID artifact.ArtifactID, artifactName string) error {
	return nil
}

func (d *DotProgress) BuildArtifactCompleted(artifactID artifact.ArtifactID, artifactName, logURI string, cachedBuild bool) error {
	return nil
}

func (d *DotProgress) BuildArtifactFailure(artifactID artifact.ArtifactID, artifactName, logURI string, errorMessage string, cachedBuild bool) error {
	return nil
}

func (d *DotProgress) BuildArtifactProgress(artifactID artifact.ArtifactID, artifactName, timeStamp, message, facility, pipeName, source string) error {
	return nil
}

func (d *DotProgress) StillBuilding(numCompleted, numTotal int) error { return nil }

func (d *DotProgress) ArtifactStepStarted(artifactID artifact.ArtifactID, artifactName, step string, counter int64, counterCountsBytes bool) error {
	return nil
}

func (d *DotProgress) ArtifactStepIncrement(artifactID artifact.ArtifactID, artifactName, step string, increment int64) error {
	return nil
}

func (d *DotProgress) ArtifactStepCompleted(artifactID artifact.ArtifactID, artifactname, step string) error {
	return nil
}

func (d *DotProgress) ArtifactStepFailure(artifactID artifact.ArtifactID, artifactname, step, errorMessage string) error {
	return nil
}
