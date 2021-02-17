package runtime

import (
	"sync"

	"github.com/ActiveState/cli/pkg/platform/api/buildlogstream"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/runtime2/build"
	"github.com/ActiveState/cli/pkg/platform/runtime2/build/alternative"
	"github.com/ActiveState/cli/pkg/platform/runtime2/model"
	"github.com/ActiveState/cli/pkg/project"
)

// maximum number of parallel artifact installations
const maxConcurrency = 3

// Setup provides methods to setup a fully-function runtime that *only* requires interactions with the local file system.
type Setup struct {
	client     ClientProvider
	msgHandler build.MessageHandler
}

// ClientProvider is the interface for all functions that involve backend communication
type ClientProvider interface {
	Solve() (*inventory_models.Order, error)
	Build(*inventory_models.Order) (*build.BuildResult, error)
	BuildLog(msgHandler buildlogstream.MessageHandler, recipe *inventory_models.Recipe) (model.BuildLog, error)
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
		// build is complete already, just install the artifacts
		for _, a := range artifacts {
			err := s.setupArtifact(buildResult.BuildEngine, a.ArtifactID, a.DownloadURL)
			if err != nil {
				return err
			}
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

func (s *Setup) installFromBuildLog(buildResult *build.BuildResult) error {
	// Access the build log to receive build updates.
	// Note: This may not actually connect to the build log if the build has already finished.
	buildLog, err := s.client.BuildLog(s.msgHandler, buildResult.Recipe)
	if err != nil {
		return err
	}
	defer buildLog.Wait()

	// wait for artifacts to be built and set them up in parallel with maximum concurrency
	ready := buildLog.BuiltArtifactsChannel()
	var wg sync.WaitGroup
	errCh := make(chan error)
	defer close(errCh)
	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for a := range ready {
				// setup
				err := s.setupArtifact(buildResult.BuildEngine, a.ID, a.DownloadURL)
				if err != nil {
					errCh <- err
				}
			}
		}()
	}
	wg.Wait()

	err = <-buildLog.Err()
	if err != nil {
		return err
	}

	err = <-errCh
	if err != nil {
		return err
	}

	return nil
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
