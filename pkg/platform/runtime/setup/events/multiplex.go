package events

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
)

type MultiPlexedProgress struct {
	digesters []ProgressDigester
}

func NewMultiPlexedProgress(digesters ...ProgressDigester) *MultiPlexedProgress {
	return &MultiPlexedProgress{digesters}
}

func (mp *MultiPlexedProgress) BuildStarted(totalArtifacts int64) error {
	var aggErr error
	for _, d := range mp.digesters {
		err := d.BuildStarted(totalArtifacts)
		if err != nil {
			aggErr = errs.Wrap(aggErr, "BuildStarted event error: %v", err)
		}
	}
	return aggErr
}

func (mp *MultiPlexedProgress) BuildCompleted(withFailures bool) error {
	var aggErr error
	for _, d := range mp.digesters {
		err := d.BuildCompleted(withFailures)
		if err != nil {
			aggErr = errs.Wrap(aggErr, "BuildCompleted event error: %v", err)
		}
	}
	return aggErr
}

func (mp *MultiPlexedProgress) InstallationStarted(totalArtifacts int64) error {
	var aggErr error
	for _, d := range mp.digesters {
		err := d.InstallationStarted(totalArtifacts)
		if err != nil {
			aggErr = errs.Wrap(aggErr, "InstallationStarted event error: %v", err)
		}
	}
	return aggErr
}
func (mp *MultiPlexedProgress) InstallationIncrement() error {
	var aggErr error
	for _, d := range mp.digesters {
		err := d.InstallationIncrement()
		if err != nil {
			aggErr = errs.Wrap(aggErr, "InstallationIncrement event error: %v", err)
		}
	}
	return aggErr
}
func (mp *MultiPlexedProgress) BuildArtifactStarted(artifactID artifact.ArtifactID, name string) error {
	var aggErr error
	for _, d := range mp.digesters {
		err := d.BuildArtifactStarted(artifactID, name)
		if err != nil {
			aggErr = errs.Wrap(aggErr, "BuildArtifactStarted event error: %v", err)
		}
	}
	return aggErr
}
func (mp *MultiPlexedProgress) BuildArtifactCompleted(artifactID artifact.ArtifactID, artifactName, logURI string) error {
	var aggErr error
	for _, d := range mp.digesters {
		err := d.BuildArtifactCompleted(artifactID, artifactName, logURI)
		if err != nil {
			aggErr = errs.Wrap(aggErr, "BuildArtifactCompleted event error: %v", err)
		}
	}
	return aggErr
}
func (mp *MultiPlexedProgress) BuildArtifactFailure(artifactID artifact.ArtifactID, artifactName, logURI string, errMsg string) error {
	var aggErr error
	for _, d := range mp.digesters {
		err := d.BuildArtifactFailure(artifactID, artifactName, logURI, errMsg)
		if err != nil {
			aggErr = errs.Wrap(aggErr, "BuildArtifactFailure event error: %v", err)
		}
	}
	return aggErr
}

func (mp *MultiPlexedProgress) ArtifactStepStarted(artifactID artifact.ArtifactID, artifactName, step string, counter int64, counterCountsBytes bool) error {
	var aggErr error
	for _, d := range mp.digesters {
		err := d.ArtifactStepStarted(artifactID, artifactName, step, counter, counterCountsBytes)
		if err != nil {
			aggErr = errs.Wrap(aggErr, "ArtifactStepStarted event error: %v", err)
		}
	}
	return aggErr
}

func (mp *MultiPlexedProgress) ArtifactStepIncrement(artifactID artifact.ArtifactID, artifactName, step string, increment int64) error {
	var aggErr error
	for _, d := range mp.digesters {
		err := d.ArtifactStepIncrement(artifactID, artifactName, step, increment)
		if err != nil {
			aggErr = errs.Wrap(aggErr, "ArtifactStepIncrement event error: %v", err)
		}
	}
	return aggErr
}

func (mp *MultiPlexedProgress) ArtifactStepCompleted(artifactID artifact.ArtifactID, artifactName, step string) error {
	var aggErr error
	for _, d := range mp.digesters {
		err := d.ArtifactStepCompleted(artifactID, artifactName, step)
		if err != nil {
			aggErr = errs.Wrap(aggErr, "ArtifactStepCompleted event error: %v", err)
		}
	}
	return aggErr
}

func (mp *MultiPlexedProgress) ArtifactStepFailure(artifactID artifact.ArtifactID, artifactName, step, errorMessage string) error {
	var aggErr error
	for _, d := range mp.digesters {
		err := d.ArtifactStepFailure(artifactID, artifactName, step, errorMessage)
		if err != nil {
			aggErr = errs.Wrap(aggErr, "ArtifactStepFailure event error: %v", err)
		}
	}
	return aggErr
}

func (mp *MultiPlexedProgress) Close() error {
	var aggErr error
	for _, d := range mp.digesters {
		err := d.Close()
		if err != nil {
			aggErr = errs.Wrap(aggErr, "Could not close ProgressDigester: %v", err)
		}
	}
	return aggErr
}
