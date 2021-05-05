package setup

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ActiveState/cli/pkg/platform/runtime/executor"
	"github.com/gammazero/workerpool"
	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/analytics"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/download"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/proxyreader"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/platform/api/buildlogstream"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	apimodel "github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/envdef"
	"github.com/ActiveState/cli/pkg/platform/runtime/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/buildlog"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/implementations/alternative"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/implementations/camel"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/faiface/mainthread"
)

// MaxConcurrency is maximum number of parallel artifact installations
const MaxConcurrency = 10

// NotInstalledError is an error returned when the runtime is not completely installed yet.
var NotInstalledError = errs.New("Runtime is not completely installed.")

// ArtifactSetupErrors combines all errors that can happen while installing artifacts in parallel
type ArtifactSetupErrors struct {
	errs []error
}

func (a *ArtifactSetupErrors) Error() string {
	var errors []string
	for _, err := range a.errs {
		errors = append(errors, errs.Join(err, " :: ").Error())
	}
	return "Not all artifacts could be installed, errors:\n" + strings.Join(errors, "\n")
}

// Errors returns the individual error messages collected from all failing artifact installations
func (a *ArtifactSetupErrors) Errors() []error {
	return a.errs
}

// UserError returns a message including all user-facing sub-error messages
func (a *ArtifactSetupErrors) UserError() string {
	var errStrings []string
	for _, err := range a.errs {
		errStrings = append(errStrings, locale.JoinErrors(err, " :: ").UserError())
	}
	return locale.Tl("setup_artifacts_err", "Not all artifacts could be installed:\n{{.V0}}", strings.Join(errStrings, "\n"))
}

// Events is the interface for callback functions that are called during
// runtime set-up when progress messages can be forwarded to the user
type Events interface {
	buildlog.Events

	// ChangeSummary summarizes the changes to the current project during the InstallRuntime() call.
	// This summary is printed as soon as possible, providing the State Tool user with an idea of the complexity of the requested build.
	// The arguments are for the changes introduced in the latest commit that this Setup is setting up.
	ChangeSummary(artifacts map[artifact.ArtifactID]artifact.ArtifactRecipe, requested artifact.ArtifactChangeset, changed artifact.ArtifactChangeset)
	TotalArtifacts(total int)
	ArtifactStepStarting(events.SetupStep, artifact.ArtifactID, string, int)
	ArtifactStepProgress(events.SetupStep, artifact.ArtifactID, int)
	ArtifactStepCompleted(events.SetupStep, artifact.ArtifactID)
	ArtifactStepFailed(events.SetupStep, artifact.ArtifactID, string)
}

type Targeter interface {
	CommitUUID() strfmt.UUID
	Name() string
	Owner() string
	Dir() string

	// OnlyUseCache communicates that this target should only use cached runtime information (ie. don't check for updates)
	OnlyUseCache() bool
}

// Setup provides methods to setup a fully-function runtime that *only* requires interactions with the local file system.
type Setup struct {
	model  ModelProvider
	target Targeter
	events Events
	store  *store.Store
}

// ModelProvider is the interface for all functions that involve backend communication
type ModelProvider interface {
	ResolveRecipe(commitID strfmt.UUID, owner, projectName string) (*inventory_models.Recipe, error)
	RequestBuild(recipeID, commitID strfmt.UUID, owner, project string) (headchef.BuildStatusEnum, *headchef_models.BuildStatusResponse, error)
	FetchBuildResult(commitID strfmt.UUID, owner, project string) (*model.BuildResult, error)
	SignS3URL(uri *url.URL) (*url.URL, error)
}

type Setuper interface {
	// ReusableArtifact returns artifact stores for the artifacts that are already installed and can be re-used for this setup.
	ReusableArtifacts(artifact.ArtifactChangeset, store.StoredArtifactMap) store.StoredArtifactMap
	// DeleteOutdatedArtifacts deletes outdated artifact as best as it can
	DeleteOutdatedArtifacts(artifact.ArtifactChangeset, store.StoredArtifactMap, store.StoredArtifactMap) error
	ResolveArtifactName(artifact.ArtifactID) string
	DownloadsFromBuild(buildStatus *headchef_models.BuildStatusResponse) ([]artifact.ArtifactDownload, error)
}

// ArtifactSetuper is the interface for an implementation of artifact setup functions
// These need to be specialized for each BuildEngine type
type ArtifactSetuper interface {
	EnvDef(tmpInstallDir string) (*envdef.EnvironmentDefinition, error)
	Unarchiver() unarchiver.Unarchiver
}

// New returns a new Setup instance that can install a Runtime locally on the machine.
func New(target Targeter, msgHandler Events) *Setup {
	return NewWithModel(target, msgHandler, model.NewDefault())
}

// NewWithModel returns a new Setup instance with a customized model eg., for testing purposes
func NewWithModel(target Targeter, msgHandler Events, model ModelProvider) *Setup {
	return &Setup{model, target, msgHandler, nil}
}

// Update installs the runtime locally (or updates it if it's already partially installed)
func (s *Setup) Update() error {
	err := s.update()
	if err != nil {
		analytics.EventWithLabel(analytics.CatRuntime, analytics.ActRuntimeFailure, analytics.LblRtFailUpdate)
		return err
	}
	return nil
}

func (s *Setup) update() error {
	// Request build
	buildResult, err := s.model.FetchBuildResult(s.target.CommitUUID(), s.target.Owner(), s.target.Name())
	if err != nil {
		return errs.Wrap(err, "Failed to fetch build result")
	}

	if buildResult.BuildStatus == headchef.Started {
		analytics.Event(analytics.CatRuntime, analytics.ActRuntimeBuild)
		ns := project.Namespaced{
			Owner:   s.target.Owner(),
			Project: s.target.Name(),
		}
		analytics.EventWithLabel(analytics.CatRuntime, analytics.ActBuildProject, ns.String())
	}

	// Compute and handle the change summary
	artifacts := artifact.NewMapFromRecipe(buildResult.Recipe)

	s.store = store.New(s.target.Dir())
	oldRecipe, err := s.store.Recipe()
	if err != nil {
		logging.Debug("Could not load existing recipe.  Maybe it is a new installation: %v", err)
	}
	requestedArtifacts := artifact.NewArtifactChangesetByRecipe(oldRecipe, buildResult.Recipe, true)
	changedArtifacts := artifact.NewArtifactChangesetByRecipe(oldRecipe, buildResult.Recipe, false)
	s.events.ChangeSummary(artifacts, requestedArtifacts, changedArtifacts)

	setup, err := s.selectSetupImplementation(buildResult.BuildEngine, artifacts)
	if err != nil {
		return errs.Wrap(err, "Failed to select setup implementation")
	}

	storedArtifacts, err := s.store.Artifacts()
	if err != nil {
		return locale.WrapError(err, "err_stored_artifacts", "Could not unmarshal stored artifacts, your install may be corrupted.")
	}

	alreadyInstalled := setup.ReusableArtifacts(changedArtifacts, storedArtifacts)

	err = setup.DeleteOutdatedArtifacts(changedArtifacts, storedArtifacts, alreadyInstalled)
	if err != nil {
		logging.Error("Could not delete outdated artifacts: %v, falling back to removing everything", err)
		err = os.RemoveAll(s.store.InstallPath())
		if err != nil {
			return locale.WrapError(err, "Failed to clean installation path")
		}
	}

	// only send the download analytics event, if we have to install artifacts that are not yet installed
	if len(artifacts) != len(alreadyInstalled) {
		// if we get here, we dowload artifacts
		analytics.Event(analytics.CatRuntime, analytics.ActRuntimeDownload)
	}

	err = s.installArtifacts(buildResult, artifacts, alreadyInstalled, setup)
	if err != nil {
		return err
	}

	edGlobal, err := s.store.UpdateEnviron(buildResult.OrderedArtifacts())
	if err != nil {
		return errs.Wrap(err, "Could not save combined environment file")
	}

	// Create executors
	execPath := filepath.Join(s.target.Dir(), "exec")
	if err := fileutils.MkdirUnlessExists(execPath); err != nil {
		return locale.WrapError(err, "err_deploy_execpath", "Could not create exec directory.")
	}

	exePaths, err := edGlobal.ExecutablePaths()
	if err != nil {
		return locale.WrapError(err, "err_deploy_execpaths", "Could not retrieve runtime executable paths")
	}

	exec := executor.NewWithBinPath(s.target.Dir(), execPath)
	if err := exec.Update(exePaths); err != nil {
		return locale.WrapError(err, "err_deploy_executors", "Could not create executors")
	}

	// Install PPM Shim if any of the installed artifacts provide the Perl executable
	if activePerlPath := edGlobal.FindBinPathFor(constants.ActivePerlExecutable); activePerlPath != "" {
		err = installPPMShim(activePerlPath)
		if err != nil {
			return errs.Wrap(err, "Failed to install the PPM shim command at %s", activePerlPath)
		}
	}

	// clean up temp directory
	tempDir := filepath.Join(s.store.InstallPath(), constants.LocalRuntimeTempDirectory)
	err = os.RemoveAll(tempDir)
	if err != nil {
		logging.Errorf("Failed to remove temporary installation directory %s: %v", tempDir, err)
	}

	if err := s.store.StoreRecipe(buildResult.Recipe); err != nil {
		return errs.Wrap(err, "Could not save recipe file.")
	}

	if err := s.store.MarkInstallationComplete(s.target.CommitUUID()); err != nil {
		return errs.Wrap(err, "Could not mark install as complete.")
	}

	return nil
}

func aggregateErrors() (chan<- error, <-chan error) {
	aggErr := make(chan error)
	bgErrs := make(chan error)
	go func() {
		var errs []error
		for err := range bgErrs {
			errs = append(errs, err)
		}

		if len(errs) > 0 {
			aggErr <- &ArtifactSetupErrors{errs}
		} else {
			aggErr <- nil
		}
	}()

	return bgErrs, aggErr
}

func (s *Setup) installArtifacts(buildResult *model.BuildResult, artifacts artifact.ArtifactRecipeMap, alreadyInstalled store.StoredArtifactMap, setup Setuper) error {
	if !buildResult.BuildReady && buildResult.BuildEngine == model.Camel {
		return locale.NewInputError("build_status_in_progress", "", apimodel.ProjectURL(s.target.Owner(), s.target.Name(), s.target.CommitUUID().String()))
	}
	// Artifacts are installed in two stages
	// - The first stage runs concurrently in MaxConcurrency worker threads (download, unpacking, relocation)
	// - The second stage moves all files into its final destination is running in a single thread (using the mainthread library) to avoid file conflicts

	var err error
	if buildResult.BuildReady {
		err = s.installFromBuildResult(buildResult, alreadyInstalled, setup)
	} else {
		err = s.installFromBuildLog(buildResult, artifacts, alreadyInstalled, setup)
	}

	return err
}

// setupArtifactSubmitFunction returns a function that sets up an artifact and can be submitted to a workerpool
func (s *Setup) setupArtifactSubmitFunction(a artifact.ArtifactDownload, buildResult *model.BuildResult, setup Setuper, errors chan<- error) func() {
	return func() {
		// This is the name used to describe the artifact.  As camel bundles all artifacts in one tarball, we call it 'bundle'
		name := setup.ResolveArtifactName(a.ArtifactID)
		if err := s.setupArtifact(buildResult.BuildEngine, a.ArtifactID, a.UnsignedURI, name); err != nil {
			if err != nil {
				errors <- locale.WrapError(err, "artifact_setup_failed", "", name, a.ArtifactID.String())
			}
		}
	}
}

func (s *Setup) installFromBuildResult(buildResult *model.BuildResult, alreadyInstalled store.StoredArtifactMap, setup Setuper) error {
	downloads, err := setup.DownloadsFromBuild(buildResult.BuildStatusResponse)
	if err != nil {
		return errs.Wrap(err, "Could not fetch artifacts to download.")
	}
	s.events.TotalArtifacts(len(downloads) - len(alreadyInstalled))

	errs, aggregatedErr := aggregateErrors()
	mainthread.Run(func() {
		defer close(errs)
		wp := workerpool.New(MaxConcurrency)
		for _, a := range downloads {
			if _, ok := alreadyInstalled[a.ArtifactID]; ok {
				continue
			}
			wp.Submit(s.setupArtifactSubmitFunction(a, buildResult, setup, errs))
		}

		wp.StopWait()
	})

	return <-aggregatedErr
}

func (s *Setup) installFromBuildLog(buildResult *model.BuildResult, artifacts artifact.ArtifactRecipeMap, alreadyInstalled store.StoredArtifactMap, setup Setuper) error {
	s.events.TotalArtifacts(len(artifacts) - len(alreadyInstalled))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, err := buildlogstream.Connect(ctx)
	if err != nil {
		return errs.Wrap(err, "Could not get build updates")
	}
	defer conn.Close()

	buildLog, err := buildlog.New(artifacts, conn, s.events, *buildResult.Recipe.RecipeID)

	errs, aggregatedErr := aggregateErrors()

	mainthread.Run(func() {
		defer close(errs)

		var wg sync.WaitGroup
		defer wg.Wait()
		wg.Add(1)
		go func() {
			// wp.StopWait needs to be run in this go-routine after ALL tasks are scheduled, hence we need to add an extra wait group
			defer wg.Done()
			wp := workerpool.New(MaxConcurrency)
			defer wp.StopWait()

			for a := range buildLog.BuiltArtifactsChannel() {
				if _, ok := alreadyInstalled[a.ArtifactID]; ok {
					continue
				}
				wp.Submit(s.setupArtifactSubmitFunction(a, buildResult, setup, errs))
			}
		}()

		if err = buildLog.Wait(); err != nil {
			errs <- err
		}
	})

	return <-aggregatedErr
}

// setupArtifact sets up an individual artifact
// The artifact is downloaded, unpacked and then processed by the artifact setup implementation
func (s *Setup) setupArtifact(buildEngine model.BuildEngine, a artifact.ArtifactID, unsignedURI, artifactName string) error {
	as, err := s.selectArtifactSetupImplementation(buildEngine, a)
	if err != nil {
		return errs.Wrap(err, "Failed to select artifact setup implementation")
	}

	targetDir := filepath.Join(s.store.InstallPath(), constants.LocalRuntimeTempDirectory)
	if err := fileutils.MkdirUnlessExists(targetDir); err != nil {
		return errs.Wrap(err, "Could not create temp runtime dir")
	}

	unarchiver := as.Unarchiver()
	archivePath := filepath.Join(targetDir, a.String()+unarchiver.Ext())
	downloadProgress := events.NewIncrementalProgress(s.events, events.Download, a, artifactName)
	if err := s.downloadArtifact(unsignedURI, archivePath, downloadProgress); err != nil {
		err := errs.Wrap(err, "Could not download artifact %s", unsignedURI)
		s.events.ArtifactStepFailed(events.Download, a, err.Error())
		return err
	}
	s.events.ArtifactStepCompleted(events.Download, a)

	unpackedDir := filepath.Join(targetDir, a.String())
	logging.Debug("Unarchiving %s (%s) to %s", archivePath, unsignedURI, unpackedDir)

	// ensure that the unpack dir is empty
	err = os.RemoveAll(unpackedDir)
	if err != nil {
		return errs.Wrap(err, "Could not remove previous temporary installation directory.")
	}

	unpackProgress := events.NewIncrementalProgress(s.events, events.Unpack, a, artifactName)
	numFiles, err := s.unpackArtifact(unarchiver, archivePath, unpackedDir, unpackProgress)
	if err != nil {
		err := errs.Wrap(err, "Could not unpack artifact %s", archivePath)
		s.events.ArtifactStepFailed(events.Unpack, a, err.Error())
		return err
	}
	s.events.ArtifactStepCompleted(events.Unpack, a)

	envDef, err := as.EnvDef(unpackedDir)
	if err != nil {
		return errs.Wrap(err, "Could not collect env info for artifact")
	}

	cnst := envdef.NewConstants(s.store.InstallPath())
	envDef = envDef.ExpandVariables(cnst)
	err = envDef.ApplyFileTransforms(filepath.Join(unpackedDir, envDef.InstallDir), cnst)
	if err != nil {
		return locale.WrapError(err, "runtime_alternative_file_transforms_err", "", "Could not apply necessary file transformations after unpacking")
	}

	// move files to installation path in main thread, such that file operations are synchronized
	return mainthread.CallErr(func() error { return s.moveToInstallPath(a, artifactName, unpackedDir, envDef, numFiles) })
}

func (s *Setup) moveToInstallPath(a artifact.ArtifactID, artifactName string, unpackedDir string, envDef *envdef.EnvironmentDefinition, numFiles int) error {
	// clean up the unpacked dir
	defer os.RemoveAll(unpackedDir)

	var files []string
	var dirs []string
	onMoveFile := func(fromPath, toPath string) {
		if fileutils.IsDir(toPath) {
			dirs = append(dirs, toPath)
		} else {
			files = append(files, toPath)
		}
		s.events.ArtifactStepProgress(events.Install, a, 1)
	}
	s.events.ArtifactStepStarting(events.Install, a, artifactName, numFiles)
	err := fileutils.MoveAllFilesRecursively(
		filepath.Join(unpackedDir, envDef.InstallDir),
		s.store.InstallPath(), onMoveFile,
	)
	if err != nil {
		err := errs.Wrap(err, "Move artifact failed")
		s.events.ArtifactStepFailed(events.Install, a, err.Error())
		return err
	}
	s.events.ArtifactStepCompleted(events.Install, a)

	if err := s.store.StoreArtifact(store.NewStoredArtifact(a, files, dirs, envDef)); err != nil {
		return errs.Wrap(err, "Could not store artifact meta info")
	}

	return nil
}

// downloadArtifact retrieves the tarball for an artifactID
// Note: the tarball may also be retrieved from a local cache directory if that is available.
func (s *Setup) downloadArtifact(unsignedURI string, targetFile string, progress *events.IncrementalProgress) error {
	artifactURL, err := url.Parse(unsignedURI)
	if err != nil {
		return errs.Wrap(err, "Could not parse artifact URL %s.", unsignedURI)
	}

	downloadURL, err := s.model.SignS3URL(artifactURL)
	if err != nil {
		return errs.Wrap(err, "Could not sign artifact URL %s.", unsignedURI)
	}

	b, err := download.GetWithProgress(downloadURL.String(), progress)
	if err != nil {
		return errs.Wrap(err, "Download %s failed", downloadURL)
	}
	if err := fileutils.WriteFile(targetFile, b); err != nil {
		return errs.Wrap(err, "Writing download to target file %s failed", targetFile)
	}
	return nil
}

func (s *Setup) unpackArtifact(ua unarchiver.Unarchiver, tarballPath string, targetDir string, progress *events.IncrementalProgress) (int, error) {
	f, i, err := ua.PrepareUnpacking(tarballPath, targetDir)
	progress.TotalSize(int(i))
	defer f.Close()
	if err != nil {
		return 0, errs.Wrap(err, "Prepare for unpacking failed")
	}
	var numUnpackedFiles int
	ua.SetNotifier(func(_ string, _ int64, isDir bool) {
		if !isDir {
			numUnpackedFiles++
		}
	})
	proxy := proxyreader.NewProxyReader(progress, f)
	return numUnpackedFiles, ua.Unarchive(proxy, i, targetDir)
}

func (s *Setup) selectSetupImplementation(buildEngine model.BuildEngine, artifacts artifact.ArtifactRecipeMap) (Setuper, error) {
	switch buildEngine {
	case model.Alternative:
		return alternative.NewSetup(s.store, artifacts), nil
	case model.Camel:
		return camel.NewSetup(s.store), nil
	default:
		return nil, errs.New("Unknown build engine: %s", buildEngine)
	}
}
func (s *Setup) selectArtifactSetupImplementation(buildEngine model.BuildEngine, a artifact.ArtifactID) (ArtifactSetuper, error) {
	switch buildEngine {
	case model.Alternative:
		return alternative.NewArtifactSetup(a, s.store), nil
	case model.Camel:
		return camel.NewArtifactSetup(a, s.store), nil
	default:
		return nil, errs.New("Unknown build engine: %s", buildEngine)
	}
}
