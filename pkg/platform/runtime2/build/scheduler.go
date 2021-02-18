package build

import (
	"context"
)

type ArtifactScheduler struct {
	ctx   context.Context
	ch    chan Artifact
	errCh chan error
}

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

func (as *ArtifactScheduler) Wait() error {
	err := <-as.errCh
	close(as.ch)
	close(as.errCh)
	return err
}

func (as *ArtifactScheduler) BuiltArtifactsChannel() <-chan Artifact {
	return as.ch
}
