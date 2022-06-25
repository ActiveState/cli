package setup

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ActiveState/cli/internal/analytics"
	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/download"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/proxyreader"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	apimodel "github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifactcache"
	"github.com/ActiveState/cli/pkg/platform/runtime/envdef"
	"github.com/ActiveState/cli/pkg/platform/runtime/executor"
	"github.com/ActiveState/cli/pkg/platform/runtime/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/buildlog"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/implementations/alternative"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/implementations/camel"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/faiface/mainthread"
	"github.com/gammazero/workerpool"
	"github.com/go-openapi/strfmt"
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
	ArtifactStepStarting(events.SetupStep, artifact.ArtifactID, int)
	ArtifactStepProgress(events.SetupStep, artifact.ArtifactID, int)
	ArtifactStepCompleted(events.SetupStep, artifact.ArtifactID)
	ArtifactStepFailed(events.SetupStep, artifact.ArtifactID, string)
	SolverError(*apimodel.SolverError)
	SolverStart()
	SolverSuccess()

	ParsedArtifacts(artifactResolver events.ArtifactResolver, downloadable []artifact.ArtifactDownload, artifactIDs []artifact.FailedArtifact)
}

type Targeter interface {
	CommitUUID() strfmt.UUID
	Name() string
	Owner() string
	Dir() string
	Headless() bool
	Trigger() target.Trigger

	// ReadOnly communicates that this target should only use cached runtime information (ie. don't check for updates)
	ReadOnly() bool
	// InstallFromDir communicates that this target should only install artifacts from the given directory (i.e. offline installer)
	InstallFromDir() *string
}

type Setup struct {
	model         ModelProvider
	target        Targeter
	events        Events
	store         *store.Store
	analytics     analytics.Dispatcher
	artifactCache *artifactcache.ArtifactCache
}

// ModelProvider is the interface for all functions that involve backend communication
type ModelProvider interface {
	ResolveRecipe(commitID strfmt.UUID, owner, projectName string) (*inventory_models.Recipe, error)
	RequestBuild(recipeID, commitID strfmt.UUID, owner, project string) (headchef.BuildStatusEnum, *headchef_models.V1BuildStatusResponse, error)
	FetchBuildResult(commitID strfmt.UUID, owner, project string) (*model.BuildResult, error)
	SignS3URL(uri *url.URL) (*url.URL, error)
}

type Setuper interface {
	// DeleteOutdatedArtifacts deletes outdated artifact as best as it can
	DeleteOutdatedArtifacts(artifact.ArtifactChangeset, store.StoredArtifactMap, store.StoredArtifactMap) error
	ResolveArtifactName(artifact.ArtifactID) string
	DownloadsFromBuild(buildStatus *headchef_models.V1BuildStatusResponse) ([]artifact.ArtifactDownload, error)
}

// ArtifactSetuper is the interface for an implementation of artifact setup functions
// These need to be specialized for each BuildEngine type
type ArtifactSetuper interface {
	EnvDef(tmpInstallDir string) (*envdef.EnvironmentDefinition, error)
	Unarchiver() unarchiver.Unarchiver
}

type artifactInstaller func(artifact.ArtifactID, string, ArtifactSetuper) error

// New returns a new Setup instance that can install a Runtime locally on the machine.
func New(target Targeter, msgHandler Events, auth *authentication.Auth, an analytics.Dispatcher) *Setup {
	return NewWithModel(target, msgHandler, model.NewDefault(auth), an)
}

// NewWithModel returns a new Setup instance with a customized model eg., for testing purposes
func NewWithModel(target Targeter, msgHandler Events, model ModelProvider, an analytics.Dispatcher) *Setup {
	cache, err := artifactcache.New()
	if err != nil {
		multilog.Error("Could not create artifact cache: %v", err)
	}
	return &Setup{model, target, msgHandler, store.New(target.Dir()), an, cache}
}

// Update installs the runtime locally (or updates it if it's already partially installed)
func (s *Setup) Update(env map[string]string) error {
	// Update all the runtime artifacts
	artifacts, err := s.updateArtifacts()
	if err != nil {
		return errs.Wrap(err, "Failed to update artifacts")
	}

	// Update executors
	if err := s.updateExecutors(env, artifacts); err != nil {
		return errs.Wrap(err, "Failed to update executors")
	}

	// Mark installation as completed
	if err := s.store.MarkInstallationComplete(s.target.CommitUUID()); err != nil {
		return errs.Wrap(err, "Could not mark install as complete.")
	}

	return nil
}

func (s *Setup) updateArtifacts() ([]artifact.ArtifactID, error) {
	mutex := &sync.Mutex{}

	// Fetch and install each runtime artifact.
	artifacts, err := s.fetchAndInstallArtifacts(func(a artifact.ArtifactID, archivePath string, as ArtifactSetuper) error {
		// Set up target and unpack directories
		targetDir := filepath.Join(s.store.InstallPath(), constants.LocalRuntimeTempDirectory)
		if err := fileutils.MkdirUnlessExists(targetDir); err != nil {
			return errs.Wrap(err, "Could not create temp runtime dir")
		}
		unpackedDir := filepath.Join(targetDir, a.String())

		logging.Debug("Unarchiving %s to %s", archivePath, unpackedDir)

		// ensure that the unpack dir is empty
		err := os.RemoveAll(unpackedDir)
		if err != nil {
			return errs.Wrap(err, "Could not remove previous temporary installation directory.")
		}

		// Unpack artifact archive
		unpackProgress := events.NewIncrementalProgress(s.events, events.Unpack, a)
		numFiles, err := s.unpackArtifact(as.Unarchiver(), archivePath, unpackedDir, unpackProgress)
		if err != nil {
			err := errs.Wrap(err, "Could not unpack artifact %s", archivePath)
			s.events.ArtifactStepFailed(events.Unpack, a, err.Error())
			return err
		}
		s.events.ArtifactStepCompleted(events.Unpack, a)

		// Set up constants used to expand environment definitions
		cnst, err := envdef.NewConstants(s.store.InstallPath())
		if err != nil {
			return errs.Wrap(err, "Could not get new environment constants")
		}

		// Retrieve environment definitions for artifact
		envDef, err := as.EnvDef(unpackedDir)
		if err != nil {
			return errs.Wrap(err, "Could not collect env info for artifact")
		}

		// Expand environment definitions using constants
		envDef = envDef.ExpandVariables(cnst)
		err = envDef.ApplyFileTransforms(filepath.Join(unpackedDir, envDef.InstallDir), cnst)
		if err != nil {
			return locale.WrapError(err, "runtime_alternative_file_transforms_err", "", "Could not apply necessary file transformations after unpacking")
		}

		// Move files to installation path, ensuring file operations are synchronized
		mutex.Lock()
		err = s.moveToInstallPath(a, unpackedDir, envDef, numFiles)
		mutex.Unlock()

		return err
	})
	if err != nil {
		return artifacts, errs.Wrap(err, "Error setting up runtime")
	}

	return artifacts, nil
}

func (s *Setup) updateExecutors(env map[string]string, artifacts []artifact.ArtifactID) error {
	execPath := ExecDir(s.target.Dir())
	if err := fileutils.MkdirUnlessExists(execPath); err != nil {
		return locale.WrapError(err, "err_deploy_execpath", "Could not create exec directory.")
	}

	edGlobal, err := s.store.UpdateEnviron(artifacts)
	if err != nil {
		return errs.Wrap(err, "Could not save combined environment file")
	}

	exePaths, err := edGlobal.ExecutablePaths()
	if err != nil {
		return locale.WrapError(err, "err_deploy_execpaths", "Could not retrieve runtime executable paths")
	}

	sockPath := svcctl.NewIPCSockPathFromGlobals().String()

	exec := executor.NewWithBinPath(s.target.Dir(), execPath)
	if err := exec.Update(sockPath, env, exePaths); err != nil {
		return locale.WrapError(err, "err_deploy_executors", "Could not create executors")
	}

	return nil
}

// fetchAndInstallArtifacts returns all artifacts needed by the runtime, even if some or
// all of them were already installed.
func (s *Setup) fetchAndInstallArtifacts(installFunc artifactInstaller) ([]artifact.ArtifactID, error) {
	if s.target.InstallFromDir() != nil {
		return s.fetchAndInstallArtifactsFromDir(installFunc)
	}
	return s.fetchAndInstallArtifactsFromRecipe(installFunc)
}

func (s *Setup) fetchAndInstallArtifactsFromRecipe(installFunc artifactInstaller) ([]artifact.ArtifactID, error) {
	// Request build
	s.events.SolverStart()
	buildResult, err := s.model.FetchBuildResult(s.target.CommitUUID(), s.target.Owner(), s.target.Name())
	if err != nil {
		serr := &apimodel.SolverError{}
		if errors.As(err, &serr) {
			s.events.SolverError(serr)
			return nil, formatSolverError(serr)
		}
		return nil, errs.Wrap(err, "Failed to fetch build result")
	}

	s.events.SolverSuccess()

	// Compute and handle the change summary
	artifacts := artifact.NewMapFromRecipe(buildResult.Recipe)
	setup, err := s.selectSetupImplementation(buildResult.BuildEngine, artifacts)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to select setup implementation")
	}

	downloads, err := setup.DownloadsFromBuild(buildResult.BuildStatusResponse)
	if err != nil {
		if errors.Is(err, artifact.CamelRuntimeBuilding) {
			localeID := "build_status_in_progress"
			messageURL := apimodel.ProjectURL(s.target.Owner(), s.target.Name(), s.target.CommitUUID().String())
			if s.target.Owner() == "" && s.target.Name() == "" {
				localeID = "build_status_in_progress_headless"
				messageURL = apimodel.CommitURL(s.target.CommitUUID().String())
			}
			return nil, locale.WrapInputError(err, localeID, "", messageURL)
		}
		return nil, errs.Wrap(err, "could not extract artifacts that are ready to download.")
	}

	failedArtifacts := artifact.NewFailedArtifactsFromBuild(buildResult.BuildStatusResponse)

	s.events.ParsedArtifacts(setup.ResolveArtifactName, downloads, failedArtifacts)

	// Analytics data to send.
	dimensions := &dimensions.Values{
		CommitID: p.StrP(s.target.CommitUUID().String()),
	}

	// send analytics build event, if a new runtime has to be built in the cloud
	if buildResult.BuildStatus == headchef.Started {
		s.analytics.Event(anaConsts.CatRuntime, anaConsts.ActRuntimeBuild, dimensions)
		ns := project.Namespaced{
			Owner:   s.target.Owner(),
			Project: s.target.Name(),
		}
		s.analytics.EventWithLabel(anaConsts.CatRuntime, anaConsts.ActBuildProject, ns.String(), dimensions)
	}

	if buildResult.BuildStatus == headchef.Failed {
		s.events.BuildFinished()
		return nil, locale.NewError("headchef_build_failure", "Build Failed: {{.V0}}", buildResult.BuildStatusResponse.Message)
	}

	oldRecipe, err := s.store.Recipe()
	if err != nil {
		logging.Debug("Could not load existing recipe.  Maybe it is a new installation: %v", err)
	}
	requestedArtifacts := artifact.NewArtifactChangesetByRecipe(oldRecipe, buildResult.Recipe, true)
	changedArtifacts := artifact.NewArtifactChangesetByRecipe(oldRecipe, buildResult.Recipe, false)
	s.events.ChangeSummary(artifacts, requestedArtifacts, changedArtifacts)

	storedArtifacts, err := s.store.Artifacts()
	if err != nil {
		return nil, locale.WrapError(err, "err_stored_artifacts", "Could not unmarshal stored artifacts, your install may be corrupted.")
	}

	alreadyInstalled := reusableArtifacts(buildResult.BuildStatusResponse.Artifacts, storedArtifacts)

	err = setup.DeleteOutdatedArtifacts(changedArtifacts, storedArtifacts, alreadyInstalled)
	if err != nil {
		multilog.Error("Could not delete outdated artifacts: %v, falling back to removing everything", err)
		err = os.RemoveAll(s.store.InstallPath())
		if err != nil {
			return nil, locale.WrapError(err, "Failed to clean installation path")
		}
	}

	// only send the download analytics event, if we have to install artifacts that are not yet installed
	if len(artifacts) != len(alreadyInstalled) {
		// if we get here, we dowload artifacts
		s.analytics.Event(anaConsts.CatRuntime, anaConsts.ActRuntimeDownload, dimensions)
	}

	err = s.installArtifactsFromBuild(buildResult, artifacts, downloads, alreadyInstalled, setup, installFunc)
	if err != nil {
		return nil, err
	}
	err = s.artifactCache.Save()
	if err != nil {
		multilog.Error("Could not save artifact cache updates: %v", err)
	}

	// clean up temp directory
	tempDir := filepath.Join(s.store.InstallPath(), constants.LocalRuntimeTempDirectory)
	err = os.RemoveAll(tempDir)
	if err != nil {
		multilog.Log(logging.ErrorNoStacktrace, rollbar.Error)("Failed to remove temporary installation directory %s: %v", tempDir, err)
	}

	if err := s.store.StoreRecipe(buildResult.Recipe); err != nil {
		return nil, errs.Wrap(err, "Could not save recipe file.")
	}

	return buildResult.OrderedArtifacts(), nil
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

func (s *Setup) installArtifactsFromBuild(buildResult *model.BuildResult, artifacts artifact.ArtifactRecipeMap, downloads []artifact.ArtifactDownload, alreadyInstalled store.StoredArtifactMap, setup Setuper, installFunc artifactInstaller) error {
	// Artifacts are installed in two stages
	// - The first stage runs concurrently in MaxConcurrency worker threads (download, unpacking, relocation)
	// - The second stage moves all files into its final destination is running in a single thread (using the mainthread library) to avoid file conflicts

	var err error
	if buildResult.BuildReady {
		err = s.installFromBuildResult(buildResult, downloads, alreadyInstalled, setup, installFunc)
	} else {
		err = s.installFromBuildLog(buildResult, artifacts, downloads, alreadyInstalled, setup, installFunc)
	}

	return err
}

// setupArtifactSubmitFunction returns a function that sets up an artifact and can be submitted to a workerpool
func (s *Setup) setupArtifactSubmitFunction(a artifact.ArtifactDownload, buildResult *model.BuildResult, setup Setuper, installFunc artifactInstaller, errors chan<- error) func() {
	return func() {
		// If artifact has no valid download, just count it as completed and return
		if strings.HasPrefix(a.UnsignedURI, "s3://as-builds/noop/") {
			s.events.ArtifactStepStarting(events.Install, a.ArtifactID, 0)
			s.events.ArtifactStepCompleted(events.Install, a.ArtifactID)
			return
		}

		as, err := s.selectArtifactSetupImplementation(buildResult.BuildEngine, a.ArtifactID)
		if err != nil {
			errors <- errs.Wrap(err, "Failed to select artifact setup implementation")
		}

		unarchiver := as.Unarchiver()
		archivePath, err := s.downloadArtifact(a, unarchiver.Ext())
		if err != nil {
			name := setup.ResolveArtifactName(a.ArtifactID)
			errors <- locale.WrapError(err, "artifact_download_failed", "", name, a.ArtifactID.String())
		}

		err = installFunc(a.ArtifactID, archivePath, as)
		if err != nil {
			name := setup.ResolveArtifactName(a.ArtifactID)
			errors <- locale.WrapError(err, "artifact_setup_failed", "", name, a.ArtifactID.String())
		}
	}
}

func (s *Setup) installFromBuildResult(buildResult *model.BuildResult, downloads []artifact.ArtifactDownload, alreadyInstalled store.StoredArtifactMap, setup Setuper, installFunc artifactInstaller) error {
	s.events.TotalArtifacts(len(downloads) - len(alreadyInstalled))

	errs, aggregatedErr := aggregateErrors()
	mainthread.Run(func() {
		defer close(errs)
		wp := workerpool.New(MaxConcurrency)
		for _, a := range downloads {
			if _, ok := alreadyInstalled[a.ArtifactID]; ok {
				continue
			}
			wp.Submit(s.setupArtifactSubmitFunction(a, buildResult, setup, installFunc, errs))
		}

		wp.StopWait()
	})

	return <-aggregatedErr
}

func (s *Setup) installFromBuildLog(buildResult *model.BuildResult, artifacts artifact.ArtifactRecipeMap, downloads []artifact.ArtifactDownload, alreadyInstalled store.StoredArtifactMap, setup Setuper, installFunc artifactInstaller) error {
	s.events.TotalArtifacts(len(artifacts) - len(alreadyInstalled))

	alreadyBuilt := make(map[artifact.ArtifactID]struct{})
	for _, d := range downloads {
		alreadyBuilt[d.ArtifactID] = struct{}{}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	buildLog, err := buildlog.New(ctx, artifacts, alreadyBuilt, s.events, *buildResult.Recipe.RecipeID)
	defer func() {
		if err := buildLog.Close(); err != nil {
			logging.Debug("Failed to close build log: %v", errs.JoinMessage(err))
		}
	}()

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
				wp.Submit(s.setupArtifactSubmitFunction(a, buildResult, setup, installFunc, errs))
			}
		}()

		if err = buildLog.Wait(); err != nil {
			errs <- err
		}
	})

	return <-aggregatedErr
}

func (s *Setup) moveToInstallPath(a artifact.ArtifactID, unpackedDir string, envDef *envdef.EnvironmentDefinition, numFiles int) error {
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
	s.events.ArtifactStepStarting(events.Install, a, numFiles)
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

// downloadArtifactWithProgress retrieves the tarball for an artifactID
// Note: the tarball may also be retrieved from a local cache directory if that is available.
func (s *Setup) downloadArtifactWithProgress(unsignedURI string, targetFile string, progress *events.IncrementalProgress) error {
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

// downloadArtifact downloads an artifact and returns the local path to that artifact's archive.
func (s *Setup) downloadArtifact(a artifact.ArtifactDownload, extension string) (string, error) {
	if cachedPath, found := s.artifactCache.Get(a.ArtifactID); found {
		return cachedPath, nil
	}

	targetDir := filepath.Join(s.store.InstallPath(), constants.LocalRuntimeTempDirectory)
	if err := fileutils.MkdirUnlessExists(targetDir); err != nil {
		return "", errs.Wrap(err, "Could not create temp runtime dir")
	}

	archivePath := filepath.Join(targetDir, a.ArtifactID.String()+extension)
	downloadProgress := events.NewIncrementalProgress(s.events, events.Download, a.ArtifactID)
	if err := s.downloadArtifactWithProgress(a.UnsignedURI, archivePath, downloadProgress); err != nil {
		err := errs.Wrap(err, "Could not download artifact %s", a.UnsignedURI)
		s.events.ArtifactStepFailed(events.Download, a.ArtifactID, err.Error())
		return "", err
	}

	s.events.ArtifactStepCompleted(events.Download, a.ArtifactID)

	err := s.artifactCache.Store(a.ArtifactID, archivePath)
	if err != nil {
		multilog.Error("Could not store artifact in cache: %v", err)
	}

	return archivePath, nil
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

func ExecDir(targetDir string) string {
	return filepath.Join(targetDir, "exec")
}

func reusableArtifacts(requestedArtifacts []*headchef_models.V1Artifact, storedArtifacts store.StoredArtifactMap) store.StoredArtifactMap {
	keep := make(store.StoredArtifactMap)

	for _, a := range requestedArtifacts {
		if v, ok := storedArtifacts[*a.ArtifactID]; ok {
			keep[*a.ArtifactID] = v
		}
	}
	return keep
}

func formatSolverError(serr *apimodel.SolverError) error {
	var err error = serr
	// Append last five lines to error message
	offset := 0
	numLines := len(serr.ValidationErrors())
	if numLines > 5 {
		offset = numLines - 5
	}

	errorLines := strings.Join(serr.ValidationErrors()[offset:], "\n")
	// Crop at 500 characters to reduce noisy output further
	if len(errorLines) > 500 {
		offset = len(errorLines) - 499
		errorLines = fmt.Sprintf("…%s", errorLines[offset:])
	}
	isCropped := offset > 0
	croppedMessage := ""
	if isCropped {
		croppedMessage = locale.Tl("solver_err_cropped_intro", "These are the last lines of the error message:")
	}

	err = locale.WrapError(err, "solver_err", "", croppedMessage, errorLines)
	if serr.IsTransient() {
		err = errs.AddTips(serr, locale.Tr("transient_solver_tip"))
	}
	return err
}

func (s *Setup) fetchAndInstallArtifactsFromDir(installFunc artifactInstaller) ([]artifact.ArtifactID, error) {
	artifactsDir := s.target.InstallFromDir()
	if artifactsDir == nil {
		return nil, errs.New("Cannot install from a directory that is nil")
	}

	artifacts, err := fileutils.ListDir(*artifactsDir, false)
	if err != nil {
		return nil, errs.Wrap(err, "Cannot read from directory to install from")
	}
	s.events.TotalArtifacts(len(artifacts))
	logging.Debug("Found %d artifacts to install from '%s'", len(artifacts), artifactsDir)

	installedArtifacts := make([]artifact.ArtifactID, len(artifacts))

	errors, aggregatedErr := aggregateErrors()
	mainthread.Run(func() {
		defer close(errors)

		wp := workerpool.New(MaxConcurrency)

		for i, a := range artifacts {
			// Each artifact is of the form artifactID.tar.gz, so extract the artifactID from the name.
			filename := a.Path()
			extIndex := strings.Index(filename, ".")
			if extIndex == -1 {
				extIndex = len(filename)
			}
			filenameNoExt := filepath.Base(filename[0:extIndex])
			artifactID := artifact.ArtifactID(filenameNoExt)
			installedArtifacts[i] = artifactID

			// Submit the artifact for setup and install.
			wp.Submit(func() {
				as := alternative.NewArtifactSetup(artifactID, s.store) // offline installer artifacts are in this format
				err = installFunc(artifactID, filename, as)
				if err != nil {
					errors <- locale.WrapError(err, "artifact_setup_failed", "", artifactID.String(), "")
				}
			})
		}

		wp.StopWait()
	})

	return installedArtifacts, <-aggregatedErr
}
