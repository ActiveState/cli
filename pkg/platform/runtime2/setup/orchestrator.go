package setup

import (
	"os"
	"os/signal"
	"sync"

	"golang.org/x/net/context"

	"github.com/ActiveState/cli/pkg/platform/runtime2/model"
)

// orchestrateArtifactSetup handles the orchestration of setting up artifact installations in parallel
// When the ready channel indicates that a new artifact is ready to be downloaded a new
// setup task will be launched for this artifact as soon as a worker task is available.
// The number of worker tasks is limited by the constant MaxConcurrency
func orchestrateArtifactSetup(parentCtx context.Context, artifactDownloaded <-chan model.ArtifactDownload, artifactSetup func(model.ArtifactDownload) error) error {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()
	var wg sync.WaitGroup
	errCh := make(chan error)
	// Run maxConcurrency runners listening for requests from the artifactDownloaded channel
	for i := 0; i < MaxConcurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			defer cancel()
			for {
				select {
				case a, ok := <-artifactDownloaded:
					// if producer channel is closed, return
					if !ok {
						return
					}
					// Process the artifact
					err := artifactSetup(a)
					if err != nil {
						errCh <- err
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}(i)
	}
	// when all runners are down, we close the error channel to indicate to the main thread that we are done
	go func() {
		defer close(errCh)
		wg.Wait()
	}()
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// contextWithSigint returns a context that is cancelled when a sigint event is received
// This could be used to make the setup installation interruptable, but ensure that
// all go functions and resources are properly released.
// Currently, our interrupt handling mechanism simply abandons the running function.
// This works for a CLI tool, where go functions are stopped after the CLI tool finishes, but it is not a viable
// approach for a server architecture.
func contextWithSigint(ctx context.Context) (context.Context, context.CancelFunc) {
	ctxWithCancel, cancel := context.WithCancel(ctx)
	go func() {
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt)
		defer func() {
			cancel()
			signal.Stop(signalCh)
			close(signalCh)
		}()

		select {
		case <-signalCh:
		case <-ctx.Done():
		}
	}()

	return ctxWithCancel, cancel
}

// artifactScheduler provides a cancelable read-able channel of artifacts to download
// It's pretty pointless on its own and only really exists to satisfy the orchestrators use of channels that are
// necessary to make buildlog streaming work
type artifactScheduler struct {
	ctx   context.Context
	ch    chan model.ArtifactDownload
	errCh chan error
}

// newArtifactScheduler returns a new artifactScheduler scheduling the provided artifacts
func newArtifactScheduler(ctx context.Context, artifacts []model.ArtifactDownload) *artifactScheduler {
	ch := make(chan model.ArtifactDownload)
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
	return &artifactScheduler{ctx, ch, errCh}
}

// Wait waits for all artifacts to be scheduled and returns an error if the scheduling was interrupted
func (as *artifactScheduler) Wait() error {
	err := <-as.errCh
	return err
}

// BuiltArtifactsChannel returns the channel to read from
func (as *artifactScheduler) BuiltArtifactsChannel() <-chan model.ArtifactDownload {
	return as.ch
}
