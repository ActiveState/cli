package runtime

import (
	"context"
	"os"
	"os/signal"
	"sync"

	"github.com/ActiveState/cli/pkg/platform/api/buildlogstream"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/runtime2/build"
	"github.com/ActiveState/cli/pkg/platform/runtime2/build/alternative"
	"github.com/ActiveState/cli/pkg/platform/runtime2/model"
	"github.com/ActiveState/cli/pkg/project"
)

// MaxConcurrency is maximum number of parallel artifact installations
const MaxConcurrency = 3

// Setup provides methods to setup a fully-function runtime that *only* requires interactions with the local file system.
type Setup struct {
	client     ClientProvider
	msgHandler build.MessageHandler
}

// ClientProvider is the interface for all functions that involve backend communication
type ClientProvider interface {
	Solve() (*inventory_models.Order, error)
	Build(*inventory_models.Order) (*build.BuildResult, error)
	BuildLog(ctx context.Context, msgHandler buildlogstream.MessageHandler, recipe *inventory_models.Recipe) (*model.BuildLog, error)
}

// ArtifactSetuper is the interface for an implementation of artifact setup functions
// These need to be specialized for each BuildEngine type
type ArtifactSetuper interface {
	NeedsSetup() bool
	Move(tmpInstallDir string) error
	MetaDataCollection() error
	Relocate() error
}

// Setuper is the interface for an implementation of runtime setup functions
// These need to be specialized for each BuildEngine type
type Setuper interface {
	PostInstall() error
}

// NewSetup returns a new Setup instance that can install a Runtime locally on the machine.
func NewSetup(project *project.Project, msgHandler build.MessageHandler) *Setup {
	return NewSetupWithAPI(project, msgHandler, model.NewDefault())
}

// NewSetupWithAPI returns a new Setup instance with a customized API client eg., for testing purposes
func NewSetupWithAPI(project *project.Project, msgHandler build.MessageHandler, api ClientProvider) *Setup {
	panic("implement me")
}

// InstalledRuntime returns a locally installed Runtime instance.
//
// If the runtime cannot be initialized a NotInstalledError is returned.
func (s *Setup) InstalledRuntime() (Runtime, error) {
	// check if complete installation can be found locally or:
	//   return ErrNotInstalled
	// next: try to load from local installation
	panic("implement me")
}

// InstallRuntime installs the runtime locally, such that it can be retrieved with the InstalledRuntime function afterwards.
func (s *Setup) InstallRuntime() error {
	// Get order for commit
	order, err := s.client.Solve()
	if err != nil {
		return err
	}

	// Request build
	buildResult, err := s.client.Build(order)
	if err != nil {
		return err
	}

	// Compute and handle the change summary
	artifacts := build.ArtifactsFromRecipe(buildResult.Recipe)
	requestedArtifacts, changedArtifacts := s.changeSummaryArgs(buildResult)
	s.msgHandler.ChangeSummary(artifacts, requestedArtifacts, changedArtifacts)

	if build.IsBuildComplete(buildResult.Recipe) {
		err := s.installImmediately(buildResult, artifacts)
		if err != nil {
			return err
		}
	} else {
		// get artifact IDs and URLs from build log streamer
		err := s.installFromBuildLog(buildResult)
		if err != nil {
			return err
		}
	}

	// Create the setup implementation based on the build engine (alternative or camel)
	var setupImpl Setuper
	setupImpl = s.selectSetupImplementation(buildResult.BuildEngine)

	setupImpl.PostInstall()
	panic("implement me")
}

// readErrs reads the error channel until it is empty and returns the error slice
func readErrs(errCh <-chan error) []error {
	var errs []error
	for {
		select {
		case e := <-errCh:
			if e != nil {
				errs = append(errs, e)
			}
		default:
			return errs
		}
	}
}

// orchestrateArtifactSetup handles the orchestration of setting up artifact installations in parallel
// When the ready channel indicates that a new artifact is ready to be downloaded a new
// setup task will be launched for this artifact as soon as a worker task is available.
// The number of worker tasks is limited by the constant MaxConcurrency
func orchestrateArtifactSetup(parentCtx context.Context, ready <-chan build.Artifact, cb func(build.Artifact) error) error {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()
	var wg sync.WaitGroup
	errCh := make(chan error, MaxConcurrency)
	defer close(errCh)
	for i := 0; i < MaxConcurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			defer func() {
				cancel()
			}()
			for {
				select {
				case a, ok := <-ready:
					if !ok {
						return
					}
					err := cb(a)
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
	wg.Wait()

	errs := readErrs(errCh)
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// contextWithSigint returns a context that is cancelled when a sigint event is received
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

func (s *Setup) installImmediately(buildResult *build.BuildResult, artifacts map[build.ArtifactID]build.Artifact) error {
	ctx, cancel := contextWithSigint(context.Background())
	defer cancel()
	scheduler := build.NewArtifactScheduler(ctx, artifacts)

	orchErr := orchestrateArtifactSetup(ctx, scheduler.BuiltArtifactsChannel(), func(a build.Artifact) error {
		return s.setupArtifact(buildResult.BuildEngine, a.ArtifactID, a.DownloadURL)
	})

	err := scheduler.Wait()
	if err != nil {
		return err
	}

	return orchErr
}

func (s *Setup) installFromBuildLog(buildResult *build.BuildResult) error {
	ctx, cancel := contextWithSigint(context.Background())
	defer cancel()
	// Access the build log to receive build updates.
	buildLog, err := s.client.BuildLog(ctx, s.msgHandler, buildResult.Recipe)
	if err != nil {
		return err
	}

	orchErr := orchestrateArtifactSetup(ctx, buildLog.BuiltArtifactsChannel(), func(a build.Artifact) error {
		return s.setupArtifact(buildResult.BuildEngine, a.ArtifactID, a.DownloadURL)
	})

	err = buildLog.Wait()
	if err != nil {
		return err
	}

	return orchErr
}

// setupArtifact sets up artifact
// The artifact is downloaded, unpacked and then processed by the artifact setup implementation
func (s *Setup) setupArtifact(buildEngine build.BuildEngine, a build.ArtifactID, downloadURL string) error {
	as := s.selectArtifactSetupImplementation(buildEngine, a)
	if !as.NeedsSetup() {
		return nil
	}

	tarball := s.downloadArtifactTarball(a, downloadURL)
	s.msgHandler.ArtifactDownloadCompleted(string(a))

	unpackedDir := s.unpackTarball(tarball)

	as.Move(unpackedDir)
	as.MetaDataCollection()
	as.Relocate()

	panic("implement error handling")
}

func (s *Setup) changeSummaryArgs(buildResult *build.BuildResult) (requested build.ArtifactChanges, changed build.ArtifactChanges) {
	panic("implement me")
}

// downloadArtifactTarball retrieves the tarball for an artifactID
// Note: the tarball may also be retrieved from a local cache directory if that is available.
func (s *Setup) downloadArtifactTarball(artifactID build.ArtifactID, downloadURL string) string {
	s.msgHandler.ArtifactDownloadStarting("artifactName")
	panic("implement me")
}

func (s *Setup) unpackTarball(tarballPath string) string {
	panic("implement me")
}

func (s *Setup) selectSetupImplementation(buildEngine build.BuildEngine) Setuper {
	if buildEngine == build.Alternative {
		return alternative.NewSetup()
	}
	panic("implement me")
}

func (s *Setup) selectArtifactSetupImplementation(buildEngine build.BuildEngine, a build.ArtifactID) ArtifactSetuper {
	if buildEngine == build.Alternative {
		return alternative.NewArtifactSetup(a)
	}
	panic("implement me")
}
