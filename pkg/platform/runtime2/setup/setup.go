package setup

import (
	"context"

	"github.com/gammazero/workerpool"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/buildlogstream"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	apimodel "github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime2/model"
	"github.com/ActiveState/cli/pkg/platform/runtime2/setup/buildlog"
	"github.com/ActiveState/cli/pkg/platform/runtime2/setup/implementations/alternative"
	"github.com/ActiveState/cli/pkg/platform/runtime2/store"
	"github.com/ActiveState/cli/pkg/projectfile"
)

// MaxConcurrency is maximum number of parallel artifact installations
const MaxConcurrency = 3

// NotInstalledError is an error returned when the runtime is not completely installed yet.
var NotInstalledError = errs.New("Runtime is not completely installed.")

type ArtifactSetupErrors struct {
	errs []error
}

func (a *ArtifactSetupErrors) Error() string {
	return "Not all artifacts could be installed"
}

func (a *ArtifactSetupErrors) Errors() []error {
	return a.errs
}

// MessageHandler is the interface for callback functions that are called during
// runtime set-up when progress messages can be forwarded to the user
type MessageHandler interface {
	buildlog.BuildLogMessageHandler

	// ChangeSummary summarizes the changes to the current project during the InstallRuntime() call.
	// This summary is printed as soon as possible, providing the State Tool user with an idea of the complexity of the requested build.
	// The arguments are for the changes introduced in the latest commit that this Setup is setting up.
	// TODO: Decide if we want to have a method to de-activate the change summary for activations where it does not make sense.
	ChangeSummary(artifacts map[model.ArtifactID]model.Artifact, requested model.ArtifactChanges, changed model.ArtifactChanges)
	ArtifactDownloadStarting(artifactName string)
	ArtifactDownloadCompleted(artifactName string)
	ArtifactDownloadFailed(artifactName string, errorMsg string)
}

type Projecter interface {
	CommitUUID() strfmt.UUID
	Name() string
	Owner() string
	Source() *projectfile.Project
}

type Configurer interface {
	CachePath() string
}

// Setup provides methods to setup a fully-function runtime that *only* requires interactions with the local file system.
type Setup struct {
	model      ModelProvider
	project    Projecter
	config     Configurer
	msgHandler MessageHandler
}

// ModelProvider is the interface for all functions that involve backend communication
type ModelProvider interface {
	FetchCheckpointForCommit(commitID strfmt.UUID) (apimodel.Checkpoint, strfmt.DateTime, error)
	ResolveRecipe(commitID strfmt.UUID, owner, projectName string) (*inventory_models.Recipe, error)
	RequestBuild(recipeID, commitID strfmt.UUID, owner, project string) (headchef.BuildStatusEnum, *headchef_models.BuildStatusResponse, error)
	FetchBuildResult(commitID strfmt.UUID, owner, project string) (*model.BuildResult, error)
	ArtifactsFromRecipe(recipe *inventory_models.Recipe) map[model.ArtifactID]model.Artifact
	ArtifactDownloads(buildStatus *headchef_models.BuildStatusResponse) []model.ArtifactDownload
	RequestedArtifactChanges(old, new model.ArtifactMap) model.ArtifactChanges
	ResolvedArtifactChanges(old, new model.ArtifactMap) model.ArtifactChanges
	DetectArtifactChanges(oldRecipe *inventory_models.Recipe, buildResult *model.BuildResult) (requested model.ArtifactChanges, changed model.ArtifactChanges)
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

// New returns a new Setup instance that can install a Runtime locally on the machine.
func New(project Projecter, config Configurer, msgHandler MessageHandler) *Setup {
	return NewWithModel(project, config, msgHandler, model.NewDefault())
}

// NewWithModel returns a new Setup instance with a customized model eg., for testing purposes
func NewWithModel(project Projecter, config Configurer, msgHandler MessageHandler, model ModelProvider) *Setup {
	return &Setup{model, project, config, msgHandler}
}

// Update installs the runtime locally (or updates it if it's already partially installed)
func (s *Setup) Update() error {
	// Request build
	buildResult, err := s.model.FetchBuildResult(s.project.CommitUUID(), s.project.Owner(), s.project.Name())
	if err != nil {
		return err
	}

	// Compute and handle the change summary
	artifacts := s.model.ArtifactsFromRecipe(buildResult.Recipe)

	store, err := store.New(s.project.Source().Project, s.config.CachePath())
	if err != nil {
		return errs.Wrap(err, "Could not create runtime store")
	}
	oldRecipe, err := store.Recipe()
	if err != nil {
		logging.Debug("Could not load existing recipe.  Maybe it is a new installation: %v", err)
	}
	requestedArtifacts, changedArtifacts := s.model.DetectArtifactChanges(oldRecipe, buildResult)
	s.msgHandler.ChangeSummary(artifacts, requestedArtifacts, changedArtifacts)

	// TODO: Here we should remove files from artifacts that are removed either by comparing with
	// the artifactCache (that should probably be handled by the Store), or by
	// using the `changedArtifacts`

	if buildResult.BuildReady {
		err := s.installFromBuildResult(buildResult, artifacts)
		if err != nil {
			return err
		}
	} else {
		// get artifact IDs and URLs from build log streamer
		// TODO: Here we could also report a prediction of the estimated build success to the user
		err := s.installFromBuildLog(buildResult, artifacts)
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

func (s *Setup) installFromBuildResult(buildResult *model.BuildResult, _ map[model.ArtifactID]model.Artifact) error {
	var errors []error
	wp := workerpool.New(MaxConcurrency)
	for _, a := range s.model.ArtifactDownloads(buildResult.BuildStatusResponse) {
		func(a model.ArtifactDownload) {
			wp.Submit(func() {
				if err := s.setupArtifact(buildResult.BuildEngine, a.ArtifactID, a.DownloadURI); err != nil {
					errors = append(errors, err)
				}
			})
		}(a)
	}

	wp.StopWait()

	if len(errors) > 0 {
		return &ArtifactSetupErrors{errors}
	}

	return nil
}


func (s *Setup) installFromBuildLog(buildResult *model.BuildResult, artifacts map[model.ArtifactID]model.Artifact) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := buildlogstream.Connect(ctx)
	if err != nil {
		return errs.Wrap(err, "Could not get build updates")
	}

	buildLog, err := buildlog.New(artifacts, conn, s.msgHandler, *buildResult.Recipe.RecipeID)

	var errors []error
	wp := workerpool.New(MaxConcurrency)
	for a := range buildLog.BuiltArtifactsChannel() {
		func(a model.ArtifactDownload) {
			wp.Submit(func() {
				if err := s.setupArtifact(buildResult.BuildEngine, a.ArtifactID, a.DownloadURI); err != nil {
					errors = append(errors, err)
				}
			})
		}(a)
	}

	if err = buildLog.Wait(); err != nil {
		return err
	}

	wp.StopWait()

	if len(errors) > 0 {
		return &ArtifactSetupErrors{errors}
	}

	return nil
}

// setupArtifact sets up an individual artifact
// The artifact is downloaded, unpacked and then processed by the artifact setup implementation
func (s *Setup) setupArtifact(buildEngine model.BuildEngine, a model.ArtifactID, downloadURL string) error {
	as := s.selectArtifactSetupImplementation(buildEngine, a)
	if !as.NeedsSetup() {
		return nil
	}

	tarball := s.downloadArtifactTarball(a, downloadURL)
	s.msgHandler.ArtifactDownloadCompleted(string(a))

	unpackedDir := s.unpackTarball(tarball)

	// TODO: Here we want to update the artifact cache
	// NB: Be careful of concurrency when writing to the artifact cache, perhaps one file per artifact?
	// store.WriteArtifactFiles(a, unpackDir)

	as.Move(unpackedDir)
	as.MetaDataCollection()
	as.Relocate()

	panic("implement error handling")
}

// downloadArtifactTarball retrieves the tarball for an artifactID
// Note: the tarball may also be retrieved from a local cache directory if that is available.
func (s *Setup) downloadArtifactTarball(artifactID model.ArtifactID, downloadURL string) string {
	s.msgHandler.ArtifactDownloadStarting("artifactName")
	panic("implement me")
}

func (s *Setup) unpackTarball(tarballPath string) string {
	panic("implement me")
}

func (s *Setup) selectSetupImplementation(buildEngine model.BuildEngine) Setuper {
	if buildEngine == model.Alternative {
		return alternative.NewSetup()
	}
	panic("implement me")
}

func (s *Setup) selectArtifactSetupImplementation(buildEngine model.BuildEngine, a model.ArtifactID) ArtifactSetuper {
	if buildEngine == model.Alternative {
		return alternative.NewArtifactSetup(a)
	}
	panic("implement me")
}

