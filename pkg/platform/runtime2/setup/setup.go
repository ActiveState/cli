package setup

import (
	"sync"

	runtime "github.com/ActiveState/cli/pkg/platform/runtime2"
	"github.com/ActiveState/cli/pkg/platform/runtime2/api"
	"github.com/ActiveState/cli/pkg/platform/runtime2/api/client"
	"github.com/ActiveState/cli/pkg/platform/runtime2/artifact"
	rcommon "github.com/ActiveState/cli/pkg/platform/runtime2/common"
	"github.com/ActiveState/cli/pkg/platform/runtime2/setup/alternative"
	"github.com/ActiveState/cli/pkg/platform/runtime2/setup/common"
	"github.com/ActiveState/cli/pkg/project"
)

// maximum number of parallel artifact installations
const maxConcurrency = 3

// Setup provides methods to setup a fully-function runtime that *only* requires interactions with the local file system.
type Setup struct {
	client     api.ClientProvider
	msgHandler common.MessageHandler
}

// NewSetup returns a new Setup instance that can install a Runtime locally on the machine.
func NewSetup(project *project.Project, msgHandler common.MessageHandler) *Setup {
	return NewSetupWithAPI(project, msgHandler, client.NewDefault())
}

// NewSetupWithAPI returns a new Setup instance with a customized API client eg., for testing purposes
func NewSetupWithAPI(project *project.Project, msgHandler common.MessageHandler, api api.ClientProvider) *Setup {
	panic("implement me")
}

// InstalledRuntime returns a locally installed Runtime instance.  If not available on the
func (s *Setup) InstalledRuntime() (rcommon.Runtimer, error) {
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

	// Create the setup implementation based on the build engine (alternative or camel)
	var setupImpl common.Setuper
	setupImpl = s.selectSetupImplementation(buildResult.BuildEngine)

	// Compute and handle the change summary
	artifacts := artifact.FromRecipe(buildResult.Recipe)
	requestedArtifacts, changedArtifacts := s.changeSummaryArgs(buildResult)
	s.msgHandler.ChangeSummary(artifacts, requestedArtifacts, changedArtifacts)

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
	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for a := range ready {
				// setup
				s.setupArtifact(buildResult.BuildEngine, a)
			}
		}()
	}
	wg.Wait()

	err = <-buildLog.Err()
	if err != nil {
		return err
	}

	setupImpl.PostInstall()
	panic("implement me")
}

// setupArtifact sets up artifact
// The artifact is downloaded, unpacked and then processed by the artifact setup implementation
func (s *Setup) setupArtifact(buildEngine runtime.BuildEngine, a runtime.ArtifactID) {
	as := s.selectArtifactSetupImplementation(buildEngine, a)
	if !as.NeedsSetup() {
		return
	}

	tarball := s.downloadArtifactTarball(a)

	unpackedDir := s.unpackTarball(tarball)

	as.Move(unpackedDir)
	as.MetaDataCollection()
	as.Relocate()

	panic("implement error handling")
}

func (s *Setup) changeSummaryArgs(buildResult *api.BuildResult) (requested common.ArtifactChanges, changed common.ArtifactChanges) {
	panic("implement me")
}

// downloadArtifactTarball retrieves the tarball for an artifactID
// Note: the tarball may also be retrieved from a local cache directory if that is available.
func (s *Setup) downloadArtifactTarball(artifactID runtime.ArtifactID) string {
	s.msgHandler.ArtifactDownloadStarting("artifactName")
	panic("implement me")
}

func (s *Setup) unpackTarball(tarballPath string) string {
	panic("implement me")
}

func (s *Setup) selectSetupImplementation(buildEngine runtime.BuildEngine) common.Setuper {
	if buildEngine == runtime.Alternative {
		return alternative.NewSetup()
	}
	panic("implement me")
}

func (s *Setup) selectArtifactSetupImplementation(buildEngine runtime.BuildEngine, a runtime.ArtifactID) common.ArtifactSetuper {
	if buildEngine == runtime.Alternative {
		return alternative.NewArtifactSetup(a)
	}
	panic("implement me")
}
