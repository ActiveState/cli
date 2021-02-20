package build

import (
	"context"
)

// ArtifactScheduler provides a cancelable read-able channel of artifacts to download
type ArtifactScheduler struct {
	ctx   context.Context
	ch    chan Artifact
	errCh chan error
}

// NewArtifactScheduler returns a new ArtifactScheduler scheduling the provided artifacts
func NewArtifactScheduler(ctx context.Context, artifacts map[ArtifactID]Artifact) *ArtifactScheduler {
	ch := make(chan Artifact)
	errCh := make(chan error)
	go func() {
		for _, a := range artifacts {
			select {
			case ch <- a:
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			}
		}
		errCh <- nil
	}()
	return &ArtifactScheduler{ctx, ch, errCh}
}

// Wait waits for all artifacts to be scheduled and returns an error if the scheduling was interrupted
func (as *ArtifactScheduler) Wait() error {
	err := <-as.errCh
	close(as.ch)
	close(as.errCh)
	return err
}

// BuiltArtifactsChannel returns the channel to read from
func (as *ArtifactScheduler) BuiltArtifactsChannel() <-chan Artifact {
	return as.ch
}
