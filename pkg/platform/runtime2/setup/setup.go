package setup

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/gammazero/workerpool"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/download"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/platform/api/buildlogstream"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	apimodel "github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime2/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime2/envdef"
	"github.com/ActiveState/cli/pkg/platform/runtime2/model"
	"github.com/ActiveState/cli/pkg/platform/runtime2/setup/buildlog"
	"github.com/ActiveState/cli/pkg/platform/runtime2/setup/implementations/alternative"
	"github.com/ActiveState/cli/pkg/platform/runtime2/store"
	"github.com/ActiveState/cli/pkg/project"
)

// MaxConcurrency is maximum number of parallel artifact installations
const MaxConcurrency = 10

// NotInstalledError is an error returned when the runtime is not completely installed yet.
var NotInstalledError = errs.New("Runtime is not completely installed.")

type ArtifactSetupErrors struct {
	errs []error
}

func (a *ArtifactSetupErrors) Error() string {
	var errors []string
	for _, err := range a.errs {
		errors = append(errors, errs.Join(err, " :: ").Error())
	}
	return "Not all artifacts could be installed, errors: \n" + strings.Join(errors, "\n")
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
	ChangeSummary(artifacts map[artifact.ArtifactID]artifact.ArtifactRecipe, requested artifact.ArtifactChangeset, changed artifact.ArtifactChangeset)
	ArtifactDownloadStarting(id strfmt.UUID)
	ArtifactDownloadCompleted(id strfmt.UUID)
	ArtifactDownloadFailed(id strfmt.UUID, errorMsg string)
}

type Targeter interface {
	CommitUUID() strfmt.UUID
	Name() string
	Owner() string
	Dir() string
}

// Setup provides methods to setup a fully-function runtime that *only* requires interactions with the local file system.
type Setup struct {
	model      ModelProvider
	target     Targeter
	msgHandler MessageHandler
	store      *store.Store
}

// ModelProvider is the interface for all functions that involve backend communication
type ModelProvider interface {
	FetchCheckpointForCommit(commitID strfmt.UUID) (apimodel.Checkpoint, strfmt.DateTime, error)
	ResolveRecipe(commitID strfmt.UUID, owner, projectName string) (*inventory_models.Recipe, error)
	RequestBuild(recipeID, commitID strfmt.UUID, owner, project string) (headchef.BuildStatusEnum, *headchef_models.BuildStatusResponse, error)
	FetchBuildResult(commitID strfmt.UUID, owner, project string) (*model.BuildResult, error)
	SignS3URL(uri *url.URL) (*url.URL, error)
}

// ArtifactSetuper is the interface for an implementation of artifact setup functions
// These need to be specialized for each BuildEngine type
type ArtifactSetuper interface {
	EnvDef(tmpInstallDir string) (*envdef.EnvironmentDefinition, error)
	Move(tmpInstallDir string) error
	Unarchiver() unarchiver.Unarchiver
}

// Setuper is the interface for an implementation of runtime setup functions
// These need to be specialized for each BuildEngine type
type Setuper interface {
	PostInstall() error
}

// New returns a new Setup instance that can install a Runtime locally on the machine.
func New(target Targeter, msgHandler MessageHandler) *Setup {
	return NewWithModel(target, msgHandler, model.NewDefault())
}

// NewWithModel returns a new Setup instance with a customized model eg., for testing purposes
func NewWithModel(target Targeter, msgHandler MessageHandler, model ModelProvider) *Setup {
	return &Setup{model, target, msgHandler, nil}
}

// updateStepError attaches a label to an error returned during the Update() function.
// The label will be used in the analytics event send during setup failures.
type updateStepError struct {
	wrapped error
	label   string
}

func newUpdateStepError(err error, label string) *updateStepError {
	return &updateStepError{wrapped: err, label: label}
}

func (use *updateStepError) Error() error {
	return use.wrapped
}

func (use *updateStepError) Label() string {
	return use.label
}

func (s *Setup) Update() error {
	use := s.update()
	if use != nil {
		analytics.EventWithLabel(CatRuntime, ActFailure, use.Label())
		return use.Error()
	}
	return nil
}

// Update installs the runtime locally (or updates it if it's already partially installed)
func (s *Setup) update() *updateStepError {
	// Request build
	buildResult, err := s.model.FetchBuildResult(s.target.CommitUUID(), s.target.Owner(), s.target.Name())
	if err != nil {
		return newUpdateStepError(err, LblBuildResults)
	}

	if buildResult.BuildStatus == headchef.Started {
		analytics.Event(CatRuntime, ActBuild)
		ns := project.Namespaced{
			Owner:   s.target.Owner(),
			Project: s.target.Name(),
		}
		analytics.EventWithLabel(CatRuntime, analytics.ActBuildProject, ns.String())
	}

	// Compute and handle the change summary
	artifacts := artifact.NewMapFromRecipe(buildResult.Recipe)

	s.store, err = store.New(s.target.Dir())
	if err != nil {
		return newUpdateStepError(errs.Wrap(err, "Could not create runtime store"), LblStore)
	}
	oldRecipe, err := s.store.Recipe()
	if err != nil {
		logging.Debug("Could not load existing recipe.  Maybe it is a new installation: %v", err)
	}
	requestedArtifacts := artifact.NewArtifactChangesetByRecipe(oldRecipe, buildResult.Recipe, true)
	changedArtifacts := artifact.NewArtifactChangesetByRecipe(oldRecipe, buildResult.Recipe, false)
	s.msgHandler.ChangeSummary(artifacts, requestedArtifacts, changedArtifacts)

	storedArtifacts, err := s.store.Artifacts()
	if err != nil {
		return newUpdateStepError(locale.WrapError(err, "err_stored_artifacts", "Could not unmarshal stored artifacts, your install may be corrupted."), LblStore)
	}
	s.deleteOutdatedArtifacts(changedArtifacts, storedArtifacts)

	// if we get here, we dowload artifacts
	analytics.Event(CatRuntime, ActDownload)

	if buildResult.BuildReady {
		err := s.installFromBuildResult(buildResult, artifacts)
		if err != nil {
			return newUpdateStepError(err, LblArtifacts)
		}
	} else {
		err := s.installFromBuildLog(buildResult, artifacts)
		if err != nil {
			return newUpdateStepError(err, LblArtifacts)
		}
	}

	if err := s.store.UpdateEnviron(); err != nil {
		return newUpdateStepError(errs.Wrap(err, "Could not save combined environment file"), LblEnv)
	}

	err = s.selectSetupImplementation(buildResult.BuildEngine).PostInstall()
	if err != nil {
		return newUpdateStepError(errs.Wrap(err, "PostInstall failed"), LblPostInstall)
	}

	if err := s.store.MarkInstallationComplete(s.target.CommitUUID()); err != nil {
		return newUpdateStepError(errs.Wrap(err, "Could not mark install as complete."), LblStore)
	}

	return nil
}

func (s *Setup) deleteOutdatedArtifacts(changeset artifact.ArtifactChangeset, storedArtifacted []store.StoredArtifact) error {
	del := map[strfmt.UUID]struct{}{}
	for _, upd := range changeset.Updated {
		del[upd.FromID] = struct{}{}
	}
	for _, id := range changeset.Removed {
		del[id] = struct{}{}
	}

	for _, artf := range storedArtifacted {
		if _, deleteMe := del[artf.ArtifactID]; !deleteMe {
			continue
		}

		for _, file := range artf.Files {
			if !fileutils.TargetExists(file) {
				continue // don't care it's already deleted (might have been deleted by another artifact that supplied the same file)
			}
			if err := os.Remove(file); err != nil {
				return locale.WrapError(err, "err_rm_artf", "", "Could not remove old package file at {{.V0}}.", file)
			}
		}

		if err := s.store.DeleteArtifactStore(artf.ArtifactID); err != nil {
			return errs.Wrap(err, "Could not delete artifact store")
		}
	}

	return nil
}

func (s *Setup) installFromBuildResult(buildResult *model.BuildResult, _ map[artifact.ArtifactID]artifact.ArtifactRecipe) error {
	var errors []error
	wp := workerpool.New(MaxConcurrency)
	downloads, err := artifact.NewDownloadsFromBuild(s.model, buildResult.BuildStatusResponse)
	if err != nil {
		return errs.Wrap(err, "Could not fetch artifacts to download.")
	}
	for _, a := range downloads {
		func(a artifact.ArtifactDownload) {
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

func (s *Setup) installFromBuildLog(buildResult *model.BuildResult, artifacts map[artifact.ArtifactID]artifact.ArtifactRecipe) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := buildlogstream.Connect(ctx)
	if err != nil {
		return errs.Wrap(err, "Could not get build updates")
	}
	defer conn.Close()

	buildLog, err := buildlog.New(artifacts, conn, s.msgHandler, *buildResult.Recipe.RecipeID)

	var errors []error
	wp := workerpool.New(MaxConcurrency)

	go func() {
		for a := range buildLog.BuiltArtifactsChannel() {
			func(a artifact.ArtifactDownload) {
				wp.Submit(func() {
					if err := s.setupArtifact(buildResult.BuildEngine, a.ArtifactID, a.DownloadURI); err != nil {
						errors = append(errors, err)
					}
				})
			}(a)
		}
	}()

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
func (s *Setup) setupArtifact(buildEngine model.BuildEngine, a artifact.ArtifactID, downloadURL string) error {
	as := s.selectArtifactSetupImplementation(buildEngine, a)

	targetDir := filepath.Join(s.store.InstallPath(), constants.LocalRuntimeTempDirectory)
	if err := fileutils.MkdirUnlessExists(targetDir); err != nil {
		return errs.Wrap(err, "Could not create temp runtime dir")
	}

	archivePath := filepath.Join(targetDir, a.String()+".tar.gz")
	if err := s.downloadArtifact(downloadURL, archivePath); err != nil {
		return errs.Wrap(err, "Could not download artifact %s", downloadURL)
	}
	s.msgHandler.ArtifactDownloadCompleted(a)

	unpackedDir := filepath.Join(targetDir, a.String())
	logging.Debug("Unarchiving %s (%s) to %s", archivePath, downloadURL, unpackedDir)
	err := s.unpackArtifact(as.Unarchiver(), archivePath, unpackedDir)
	if err != nil {
		return errs.Wrap(err, "Could not unpack artifact %s", archivePath)
	}

	// There might be room for performance improvement here by combining the file move step with the filename collection
	// that's happening below
	var files []string
	err = filepath.Walk(unpackedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, info.Name())
		}
		return nil
	})
	if err != nil {
		return errs.Wrap(err, "Could not read unpackedDir: %s", unpackedDir)
	}

	envDef, err := as.EnvDef(unpackedDir)
	if err != nil {
		return errs.Wrap(err, "Could not collect env info for artifact")
	}

	if err := as.Move(filepath.Join(unpackedDir, envDef.InstallDir)); err != nil {
		return errs.Wrap(err, "Move artifact failed")
	}

	if err := s.store.StoreArtifact(store.NewStoredArtifact(a, files, envDef)); err != nil {
		return errs.Wrap(err, "Could not store artifact meta info")
	}

	return nil
}

// downloadArtifact retrieves the tarball for an artifactID
// Note: the tarball may also be retrieved from a local cache directory if that is available.
func (s *Setup) downloadArtifact(downloadURL string, targetFile string) error {
	s.msgHandler.ArtifactDownloadStarting("artifactName")
	b, err := download.Get(downloadURL)
	if err != nil {
		return errs.Wrap(err, "Download %s failed", downloadURL)
	}
	if err := fileutils.WriteFile(targetFile, b); err != nil {
		return errs.Wrap(err, "Writing download to target file %s failed", targetFile)
	}
	return nil
}

func (s *Setup) unpackArtifact(ua unarchiver.Unarchiver, tarballPath string, targetDir string) error {
	f, i, err := ua.PrepareUnpacking(tarballPath, targetDir)
	defer f.Close()
	if err != nil {
		return errs.Wrap(err, "Prepare for unpacking failed")
	}
	return ua.Unarchive(f, i, targetDir)
}

func (s *Setup) selectSetupImplementation(buildEngine model.BuildEngine) Setuper {
	if buildEngine == model.Alternative {
		return alternative.NewSetup()
	}
	panic("implement me")
}

func (s *Setup) selectArtifactSetupImplementation(buildEngine model.BuildEngine, a artifact.ArtifactID) ArtifactSetuper {
	if buildEngine == model.Alternative {
		return alternative.NewArtifactSetup(a, s.store)
	}
	panic("implement me")
}
