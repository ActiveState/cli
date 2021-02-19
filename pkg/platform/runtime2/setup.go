package runtime

import (
	"context"
	"os"
	"os/signal"
	"sync"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	apimodel "github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime2/build"
	"github.com/ActiveState/cli/pkg/platform/runtime2/build/alternative"
	"github.com/ActiveState/cli/pkg/platform/runtime2/model"
	"github.com/ActiveState/cli/pkg/projectfile"
	"github.com/go-openapi/strfmt"
)

// MaxConcurrency is maximum number of parallel artifact installations
const MaxConcurrency = 3

// NotInstalledError is an error returned when the runtime is not completely installed yet.
var NotInstalledError = errs.New("Runtime is not completely installed.")

// MessageHandler is the interface for callback functions that are called during
// runtime set-up when progress messages can be forwarded to the user
type MessageHandler interface {
	build.BuildLogMessageHandler

	// ChangeSummary summarizes the changes to the current project during the InstallRuntime() call.
	// This summary is printed as soon as possible, providing the State Tool user with an idea of the complexity of the requested build.
	// The arguments are for the changes introduced in the latest commit that this Setup is setting up.
	// TODO: Decide if we want to have a method to de-activate the change summary for activations where it does not make sense.
	ChangeSummary(artifacts map[build.ArtifactID]build.Artifact, requested build.ArtifactChanges, changed build.ArtifactChanges)
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
	client     ClientProvider
	project    Projecter
	config     Configurer
	msgHandler MessageHandler
}

// ClientProvider is the interface for all functions that involve backend communication
type ClientProvider interface {
	FetchCheckpointForCommit(commitID strfmt.UUID) (apimodel.Checkpoint, strfmt.DateTime, error)
	ResolveRecipe(commitID strfmt.UUID, owner, projectName string) (*inventory_models.Recipe, error)
	RequestBuild(recipeID, commitID strfmt.UUID, owner, project string) (headchef.BuildStatusEnum, *headchef_models.BuildStatusResponse, error)
	BuildLog(context.Context, map[build.ArtifactID]build.Artifact, build.BuildLogMessageHandler, strfmt.UUID) (*build.BuildLog, error)
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
func NewSetup(project Projecter, config Configurer, msgHandler MessageHandler) *Setup {
	return NewSetupWithAPI(project, config, msgHandler, model.NewDefault())
}

// NewSetupWithAPI returns a new Setup instance with a customized API client eg., for testing purposes
func NewSetupWithAPI(project Projecter, config Configurer, msgHandler MessageHandler, api ClientProvider) *Setup {
	return &Setup{api, project, config, msgHandler}
}

// InstalledRuntime returns a locally installed Runtime instance.
//
// If the runtime cannot be initialized a NotInstalledError is returned.
func (s *Setup) InstalledRuntime() (*Runtime, error) {
	store, err := NewStore(s.project.Source().Path(), s.config.CachePath())
	if err != nil {
		return nil, errs.Wrap(err, "Could not create runtime store")
	}
	if !store.HasCompleteInstallation(s.project.CommitUUID()) {
		return nil, NotInstalledError
	}
	be, err := store.BuildEngine()
	if err != nil {
		return nil, errs.Wrap(NotInstalledError, "Failed to load build engine: %v", err)
	}
	env, err := s.selectEnvironProvider(be, store.InstallPath())
	if err != nil {
		return nil, errs.Wrap(NotInstalledError, "Failed to load the environ provider: %v", err)
	}
	return newRuntime(store, env)
}

// InstallRuntime installs the runtime locally, such that it can be retrieved with the InstalledRuntime function afterwards.
func (s *Setup) InstallRuntime() error {
	// Request build
	buildResult, err := s.FetchBuildResult(s.project.CommitUUID(), s.project.Owner(), s.project.Name())
	if err != nil {
		return err
	}

	// Compute and handle the change summary
	artifacts := build.ArtifactsFromRecipe(buildResult.Recipe)

	store, err := NewStore(s.project.Source().Project, s.config.CachePath())
	if err != nil {
		return errs.Wrap(err, "Could not create runtime store")
	}
	oldRecipe, err := store.Recipe()
	if err != nil {
		logging.Debug("Could not load existing recipe.  Maybe it is a new installation: %v", err)
	}
	requestedArtifacts, changedArtifacts := changeSummaryArgs(oldRecipe, buildResult)
	s.msgHandler.ChangeSummary(artifacts, requestedArtifacts, changedArtifacts)

	// TODO: Here we should remove files from artifacts that are removed either by comparing with
	// the artifactCache (that should probably be handled by the Store), or by
	// using the `changedArtifacts`

	if build.IsBuildComplete(buildResult) {
		err := s.installImmediately(buildResult, artifacts)
		if err != nil {
			return err
		}
	} else {
		// get artifact IDs and URLs from build log streamer
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

// ValidateCheckpoint ensures that the commitID is valid and a build can succeed
func (s *Setup) ValidateCheckpoint(commitID strfmt.UUID) error {
	if commitID == "" {
		return locale.NewInputError("setup_err_runtime_no_commitid", "A CommitID is required to install this runtime environment")
	}

	checkpoint, _, err := s.client.FetchCheckpointForCommit(commitID)
	if err != nil {
		return err
	}

	for _, change := range checkpoint {
		if apimodel.NamespaceMatch(change.Namespace, apimodel.NamespacePrePlatformMatch) {
			return locale.NewInputError("installer_err_runtime_preplatform")
		}
	}
	return nil
}

// FetchBuildResult requests a build for a resolved recipe and returns the result in a BuildResult struct
func (s *Setup) FetchBuildResult(commitID strfmt.UUID, owner, project string) (*build.BuildResult, error) {
	recipe, err := s.client.ResolveRecipe(commitID, owner, project)
	if err != nil {
		return nil, locale.WrapError(err, "setup_build_resolve_recipe_err", "Could not resolve recipe for project %s/%s#%s", owner, project, commitID.String())
	}

	bse, resp, err := s.client.RequestBuild(*recipe.RecipeID, commitID, owner, project)
	if err != nil {
		return nil, locale.WrapError(err, "headchef_build_err", "Could not request build for %s/%s#%s", owner, project, commitID.String())
	}

	engine := build.BuildEngineFromResponse(resp)

	return &build.BuildResult{
		BuildEngine:         engine,
		Recipe:              recipe,
		BuildStatusResponse: resp,
		BuildStatus:         bse,
	}, nil
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
func orchestrateArtifactSetup(parentCtx context.Context, ready <-chan build.ArtifactDownload, cb func(build.ArtifactDownload) error) error {
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

func (s *Setup) installImmediately(buildResult *build.BuildResult, artifacts map[build.ArtifactID]build.Artifact) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	downloads := build.ArtifactDownloads(buildResult.BuildStatusResponse)
	scheduler := build.NewArtifactScheduler(ctx, downloads)

	orchErr := orchestrateArtifactSetup(ctx, scheduler.BuiltArtifactsChannel(), func(a build.ArtifactDownload) error {
		return s.setupArtifact(buildResult.BuildEngine, a.ArtifactID, a.DownloadURI)
	})

	err := scheduler.Wait()
	if err != nil {
		return err
	}

	return orchErr
}

func (s *Setup) installFromBuildLog(buildResult *build.BuildResult, artifacts map[build.ArtifactID]build.Artifact) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Access the build log to receive build updates.
	buildLog, err := s.client.BuildLog(ctx, artifacts, s.msgHandler, *buildResult.Recipe.RecipeID)
	if err != nil {
		return err
	}

	orchErr := orchestrateArtifactSetup(ctx, buildLog.BuiltArtifactsChannel(), func(a build.ArtifactDownload) error {
		return s.setupArtifact(buildResult.BuildEngine, a.ArtifactID, a.DownloadURI)
	})

	err = buildLog.Wait()
	if err != nil {
		return err
	}

	return orchErr
}

// setupArtifact sets up an individual artifact
// The artifact is downloaded, unpacked and then processed by the artifact setup implementation
func (s *Setup) setupArtifact(buildEngine build.BuildEngine, a build.ArtifactID, downloadURL string) error {
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

// changeSummaryArgs computes the artifact changes between an old recipe (which can be empty) and a new recipe
func changeSummaryArgs(oldRecipe *inventory_models.Recipe, buildResult *build.BuildResult) (requested build.ArtifactChanges, changed build.ArtifactChanges) {
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

func (s *Setup) selectEnvironProvider(buildEngine build.BuildEngine, installPath string) (EnvProvider, error) {
	if buildEngine == build.Alternative {
		return alternative.New(installPath)
	}
	panic("implement me")
}
