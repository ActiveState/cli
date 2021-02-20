package model

import (
	"sync"

	"github.com/ActiveState/cli/pkg/platform/runtime2/build"
)

// BuildLog is an implementation of a build log
type BuildLog struct {
	// TODO: This is just a rough outline of how it could look like
	wg    *sync.WaitGroup
	ch    chan build.Artifact
	errCh chan error
}

// NewBuildLog creates a new instance that allows us to wait for incoming build log information
func NewBuildLog() *BuildLog {
	wg := new(sync.WaitGroup)
	ch := make(chan build.Artifact)
	errCh := make(chan error)
	wg.Add(1)
	go func() {
		defer wg.Done()
		// TODO: handle the actual build log streamer here
		// on new artifact:
		//   write to ch
		// on error:
		//   write to errCh
		// on finished:
		//   close
		// on interrupt:
		//   return
	}()
	return &BuildLog{
		wg:    wg,
		ch:    ch,
		errCh: errCh,
	}
}

// Wait waits for the build log to close because the build is done and all downloadable artifacts are here
func (bl *BuildLog) Wait() error {
	bl.wg.Wait()
	return nil
}

// Close stops the build log process, eg., due to a user interruption
func (bl *BuildLog) Close() error {
	close(bl.ch)
	close(bl.errCh)
	return nil
}

// BuiltArtifactsChannel returns the channel to listen for downloadable artifacts on
func (bl *BuildLog) BuiltArtifactsChannel() <-chan build.Artifact {
	return bl.ch
}

// Err returns errors encountered during the build logging
func (bl *BuildLog) Err() <-chan error {
	return bl.errCh
}
