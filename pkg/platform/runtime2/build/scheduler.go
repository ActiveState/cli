package build

import (
	"context"
)

// ArtifactScheduler provides a cancelable read-able channel of artifacts to download
type ArtifactScheduler struct {
	ctx   context.Context
	ch    chan ArtifactDownload
	errCh chan error
}

// NewArtifactScheduler returns a new ArtifactScheduler scheduling the provided artifacts
func NewArtifactScheduler(ctx context.Context, artifacts []ArtifactDownload) *ArtifactScheduler {
	ch := make(chan ArtifactDownload)
	errCh := make(chan error)
	go func() {
		defer close(ch)
		defer close(errCh)
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
	return err
}

// BuiltArtifactsChannel returns the channel to read from
func (as *ArtifactScheduler) BuiltArtifactsChannel() <-chan ArtifactDownload {
	return as.ch
}
