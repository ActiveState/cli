package setup

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	rt "runtime"
	"strings"
	"sync"
	"time"

	bpModel "github.com/ActiveState/cli/pkg/platform/api/graphql/model/buildplanner"
	"github.com/ActiveState/cli/pkg/platform/model"

	"github.com/ActiveState/cli/internal/analytics"
	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/httputil"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/proxyreader"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/rtutils/p"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/platform/api/headchef"
	"github.com/ActiveState/cli/pkg/platform/api/inventory/inventory_models"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	apimodel "github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifactcache"
	"github.com/ActiveState/cli/pkg/platform/runtime/buildplan"
	"github.com/ActiveState/cli/pkg/platform/runtime/envdef"
	"github.com/ActiveState/cli/pkg/platform/runtime/executors"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/buildlog"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/events/progress"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/implementations/alternative"
	"github.com/ActiveState/cli/pkg/platform/runtime/setup/implementations/camel"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/ActiveState/cli/pkg/platform/runtime/validate"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/faiface/mainthread"
	"github.com/gammazero/workerpool"
	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"
)

// MaxConcurrency is maximum number of parallel artifact installations
const MaxConcurrency = 5

// NotInstalledError is an error returned when the runtime is not completely installed yet.
var NotInstalledError = errs.New("Runtime is not completely installed.")

// ArtifactSetupErrors combines all errors that can happen while installing artifacts in parallel
type ArtifactSetupErrors struct {
	errs []error
}

func (a *ArtifactSetupErrors) Error() string {
	var errors []string
	for _, err := range a.errs {
		errors = append(errors, errs.JoinMessage(err))
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
		errStrings = append(errStrings, locale.JoinedErrorMessage(err))
	}
	return locale.Tl("setup_artifacts_err", "Not all artifacts could be installed:\n{{.V0}}", strings.Join(errStrings, "\n"))
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
	auth          *authentication.Auth
	target        Targeter
	eventHandler  events.Handler
	store         *store.Store
	analytics     analytics.Dispatcher
	artifactCache *artifactcache.ArtifactCache
}

type Setuper interface {
	// DeleteOutdatedArtifacts deletes outdated artifact as best as it can
	DeleteOutdatedArtifacts(artifact.ArtifactChangeset, store.StoredArtifactMap, store.StoredArtifactMap) error
	ResolveArtifactName(artifact.ArtifactID) string
	DownloadsFromBuild(build bpModel.Build, artifacts map[strfmt.UUID]artifact.Artifact) (download []artifact.ArtifactDownload, err error)
}

// ArtifactSetuper is the interface for an implementation of artifact setup functions
// These need to be specialized for each BuildEngine type
type ArtifactSetuper interface {
	EnvDef(tmpInstallDir string) (*envdef.EnvironmentDefinition, error)
	Unarchiver() unarchiver.Unarchiver
}

type artifactInstaller func(artifact.ArtifactID, string, ArtifactSetuper) error

// New returns a new Setup instance that can install a Runtime locally on the machine.
func New(target Targeter, eventHandler events.Handler, auth *authentication.Auth, an analytics.Dispatcher) *Setup {
	return NewWithModel(target, eventHandler, auth, an)
}

// NewWithModel returns a new Setup instance with a customized model eg., for testing purposes
func NewWithModel(target Targeter, eventHandler events.Handler, auth *authentication.Auth, an analytics.Dispatcher) *Setup {
	cache, err := artifactcache.New()
	if err != nil {
		multilog.Error("Could not create artifact cache: %v", err)
	}
	return &Setup{auth, target, eventHandler, store.New(target.Dir()), an, cache}
}

// Update installs the runtime locally (or updates it if it's already partially installed)
func (s *Setup) Update() (rerr error) {
	defer func() {
		var err error
		if rerr == nil {
			err = s.eventHandler.Handle(events.Success{})
		} else {
			err = s.eventHandler.Handle(events.Failure{})
		}
		if err != nil {
			logging.Error("Could not handle Success/Failure event: %s", errs.JoinMessage(err))
		}
	}()

	// Do not allow users to deploy runtimes to the root directory (this can easily happen in docker
	// images). Note that runtime targets are fully resolved via fileutils.ResolveUniquePath(), so
	// paths like "/." and "/opt/.." resolve to simply "/" at this time.
	if rt.GOOS != "windows" && s.target.Dir() == "/" {
		return locale.NewInputError("err_runtime_setup_root", "Cannot set up a runtime in the root directory. Please specify or run from a user-writable directory.")
	}

	// Update all the runtime artifacts
	artifacts, err := s.updateArtifacts()
	if err != nil {
		return errs.Wrap(err, "Failed to update artifacts")
	}

	// Update executors
	if err := s.updateExecutors(artifacts); err != nil {
		return errs.Wrap(err, "Failed to update executors")
	}

	// Mark installation as completed
	if err := s.store.MarkInstallationComplete(s.target.CommitUUID(), fmt.Sprintf("%s/%s", s.target.Owner(), s.target.Name())); err != nil {
		return errs.Wrap(err, "Could not mark install as complete.")
	}

	return nil
}

func (s *Setup) updateArtifacts() ([]artifact.ArtifactID, error) {
	mutex := &sync.Mutex{}

	// Fetch and install each runtime artifact.
	artifacts, err := s.fetchAndInstallArtifacts(func(a artifact.ArtifactID, archivePath string, as ArtifactSetuper) (rerr error) {
		defer func() {
			if rerr != nil {
				if err := s.eventHandler.Handle(events.ArtifactInstallFailure{a, rerr}); err != nil {
					rerr = errs.Wrap(rerr, "Could not handle ArtifactInstallFailure event: %v", errs.JoinMessage(err))
					return
				}
			}
			if err := s.eventHandler.Handle(events.ArtifactInstallSuccess{a}); err != nil {
				rerr = errs.Wrap(rerr, "Could not handle ArtifactInstallSuccess event: %v", errs.JoinMessage(err))
				return
			}
		}()

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
		numFiles, err := s.unpackArtifact(as.Unarchiver(), archivePath, unpackedDir, &progress.Report{
			ReportSizeCb: func(size int) error {
				if err := s.eventHandler.Handle(events.ArtifactInstallStarted{a, size}); err != nil {
					return errs.Wrap(err, "Could not handle ArtifactInstallStarted event")
				}
				return nil
			},
			ReportIncrementCb: func(inc int) error {
				if err := s.eventHandler.Handle(events.ArtifactInstallProgress{a, inc}); err != nil {
					return errs.Wrap(err, "Could not handle ArtifactInstallProgress event")
				}
				return nil
			},
		})
		if err != nil {
			err := errs.Wrap(err, "Could not unpack artifact %s", archivePath)
			return err
		}

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

func (s *Setup) updateExecutors(artifacts []artifact.ArtifactID) error {
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

	env, err := s.store.Environ(false)
	if err != nil {
		return locale.WrapError(err, "err_setup_get_runtime_env", "Could not retrieve runtime environment")
	}

	execInit := executors.New(execPath)
	if err := execInit.Apply(svcctl.NewIPCSockPathFromGlobals().String(), s.target, env, exePaths); err != nil {
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
	return s.fetchAndInstallArtifactsFromBuildPlan(installFunc)
}

func (s *Setup) fetchAndInstallArtifactsFromBuildPlan(installFunc artifactInstaller) ([]artifact.ArtifactID, error) {
	// Request build
	if err := s.eventHandler.Handle(events.SolveStart{}); err != nil {
		return nil, errs.Wrap(err, "Could not handle SolveStart event")
	}

	bp := model.NewBuildPlanModel(s.auth)
	buildResult, err := bp.FetchBuildResult(s.target.CommitUUID())
	if err != nil {
		serr := &model.BuildPlannerError{}
		if errors.As(err, &serr) {
			if err := s.eventHandler.Handle(events.SolveError{serr}); err != nil {
				return nil, errs.Wrap(err, "Could not handle SolveError event")
			}
			return nil, formatBuildPlanError(serr)
		}
		return nil, errs.Wrap(err, "Failed to fetch build result")
	}

	if err := s.eventHandler.Handle(events.SolveSuccess{}); err != nil {
		return nil, errs.Wrap(err, "Could not handle SolveSuccess event")
	}

	// Compute and handle the change summary
	var artifacts artifact.Map
	if buildResult.Build != nil {
		artifacts, err = buildplan.NewMapFromBuildPlan(buildResult.Build)
		if err != nil {
			return nil, errs.Wrap(err, "Failed to create artifact map from build plan")
		}
	}

	setup, err := s.selectSetupImplementation(buildResult.BuildEngine, artifacts)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to select setup implementation")
	}

	// If some artifacts were already build then we can detect whether they need to be installed ahead of time
	// Note there may still be more noop artifacts, but we won't know until they have finished building.
	noopArtifacts := map[strfmt.UUID]struct{}{}
	for _, prebuiltArtf := range buildResult.Build.Artifacts {
		if prebuiltArtf.TargetID != "" && prebuiltArtf.Status != "" &&
			prebuiltArtf.Status == bpModel.ArtifactSucceeded &&
			strings.HasPrefix(prebuiltArtf.URL, "s3://as-builds/noop/") {
			noopArtifacts[prebuiltArtf.TargetID] = struct{}{}
		}
	}

	for id := range artifacts {
		if _, noop := noopArtifacts[id]; noop {
			delete(artifacts, id)
		}
	}

	downloadablePrebuiltResults, err := setup.DownloadsFromBuild(*buildResult.Build, artifacts)
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

	// buildResult doesn't have namespace info and will happily report internal only artifacts
	downloadablePrebuiltResults = funk.Filter(downloadablePrebuiltResults, func(ad artifact.ArtifactDownload) bool {
		ar, ok := artifacts[ad.ArtifactID]
		if !ok {
			return true
		}
		return ar.Namespace != inventory_models.NamespaceCoreTypeInternal
	}).([]artifact.ArtifactDownload)

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
		return nil, locale.NewError("headchef_build_failure", "Build Failed: {{.V0}}", buildResult.BuildStatusResponse.Message)
	}

	changedArtifacts, err := buildplan.NewBaseArtifactChangesetByBuildPlan(buildResult.Build, false)
	if err != nil {
		return nil, errs.Wrap(err, "Could not compute base artifact changeset")
	}

	oldBuildPlan, err := s.store.BuildPlan()
	if err != nil {
		logging.Debug("Could not load existing build plan. Maybe it is a new installation: %v", err)
	}

	if oldBuildPlan != nil {
		changedArtifacts, err = buildplan.NewArtifactChangesetByBuildPlan(oldBuildPlan, buildResult.Build, false)
		if err != nil {
			return nil, errs.Wrap(err, "Could not compute artifact changeset")
		}
	}

	storedArtifacts, err := s.store.Artifacts()
	if err != nil {
		return nil, locale.WrapError(err, "err_stored_artifacts", "Could not unmarshal stored artifacts, your install may be corrupted.")
	}

	alreadyInstalled := reusableArtifacts(buildResult.Build.Artifacts, storedArtifacts)

	// Report resolved artifacts
	artifactIDs := []artifact.ArtifactID{}
	for _, a := range artifacts {
		artifactIDs = append(artifactIDs, a.ArtifactID)
	}

	artifactNames := artifact.ResolveArtifactNames(setup.ResolveArtifactName, artifactIDs)
	artifactNamesList := []string{}
	for _, n := range artifactNames {
		artifactNamesList = append(artifactNamesList, n)
	}
	installedList := []string{}
	for _, a := range alreadyInstalled {
		installedList = append(installedList, artifactNames[a.ArtifactID])
	}
	downloadList := []string{}
	for _, a := range downloadablePrebuiltResults {
		downloadList = append(downloadList, artifactNames[a.ArtifactID])
	}
	logging.Debug(
		"Parsed artifacts.\nBuild ready: %v\nArtifact names: %v\nAlready installed: %v\nTo Download: %v",
		buildResult.BuildReady, artifactNamesList, installedList, downloadList,
	)

	artifactsToInstall := []artifact.ArtifactID{}
	if buildResult.BuildReady {
		// If the build is already done we can just look at the downloadable artifacts as they will be a fully accurate
		// prediction of what we will be installing.
		for _, a := range downloadablePrebuiltResults {
			if _, alreadyInstalled := alreadyInstalled[a.ArtifactID]; !alreadyInstalled {
				artifactsToInstall = append(artifactsToInstall, a.ArtifactID)
			}
		}
	} else {
		// If the build is not yet complete then we have to speculate as to the artifacts that will be installed.
		// The actual number of installable artifacts may be lower than what we have here, we can only do a best effort.
		for _, a := range artifacts {
			if _, alreadyInstalled := alreadyInstalled[a.ArtifactID]; !alreadyInstalled {
				artifactsToInstall = append(artifactsToInstall, a.ArtifactID)
			}
		}
	}

	// The log file we want to use for builds
	logFilePath := logging.FilePathFor(fmt.Sprintf("build-%s.log", s.target.CommitUUID().String()+"-"+time.Now().Format("20060102150405")))

	var recipeID strfmt.UUID
	if buildResult.RecipeID != "" {
		recipeID = buildResult.RecipeID
	}

	if err := s.eventHandler.Handle(events.Start{
		RecipeID:      recipeID,
		RequiresBuild: !buildResult.BuildReady,
		ArtifactNames: artifactNames,
		LogFilePath:   logFilePath,
		ArtifactsToBuild: func() []artifact.ArtifactID {
			if !buildResult.BuildReady {
				return artifact.ArtifactIDsFromBuildPlanMap(artifacts) // This does not account for cached builds
			}
			return []artifact.ArtifactID{}
		}(),
		// Yes these have the same value; this is intentional.
		// Separating these out just allows us to be more explicit and intentional in our event handling logic.
		ArtifactsToDownload: artifactsToInstall,
		ArtifactsToInstall:  artifactsToInstall,
	}); err != nil {
		return nil, errs.Wrap(err, "Could not handle Start event")
	}

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

	err = s.installArtifactsFromBuild(buildResult, artifacts, artifact.ArtifactIDsToMap(artifactsToInstall), downloadablePrebuiltResults, alreadyInstalled, setup, installFunc, logFilePath)
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

	if err := s.store.StoreBuildPlan(buildResult.Build); err != nil {
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

func (s *Setup) installArtifactsFromBuild(buildResult *model.BuildResult, artifacts artifact.Map, artifactsToInstall map[artifact.ArtifactID]struct{}, downloads []artifact.ArtifactDownload, alreadyInstalled store.StoredArtifactMap, setup Setuper, installFunc artifactInstaller, logFilePath string) error {
	// Artifacts are installed in two stages
	// - The first stage runs concurrently in MaxConcurrency worker threads (download, unpacking, relocation)
	// - The second stage moves all files into its final destination is running in a single thread (using the mainthread library) to avoid file conflicts

	var err error
	if buildResult.BuildReady {
		if err := s.eventHandler.Handle(events.BuildSkipped{}); err != nil {
			return errs.Wrap(err, "Could not handle BuildSkipped event")
		}
		err = s.installFromBuildResult(buildResult, artifacts, downloads, alreadyInstalled, setup, installFunc)
	} else {
		err = s.installFromBuildLog(buildResult, artifacts, artifactsToInstall, alreadyInstalled, setup, installFunc, logFilePath)
	}

	return err
}

// setupArtifactSubmitFunction returns a function that sets up an artifact and can be submitted to a workerpool
func (s *Setup) setupArtifactSubmitFunction(a artifact.ArtifactDownload, ar *artifact.Artifact, expectedArtifactInstalls map[artifact.ArtifactID]struct{}, buildResult *model.BuildResult, setup Setuper, installFunc artifactInstaller, errors chan<- error) func() {
	return func() {
		// If artifact has no valid download, just count it as completed and return
		if strings.HasPrefix(a.UnsignedURI, "s3://as-builds/noop/") ||
			// Internal namespace artifacts are not to be downloaded
			(ar != nil && ar.Namespace == inventory_models.NamespaceCoreTypeInternal) {
			logging.Debug("Skipping setup of noop artifact: %s", a.ArtifactID)
			if _, expected := expectedArtifactInstalls[a.ArtifactID]; expected {
				if err := s.eventHandler.Handle(events.ArtifactDownloadSkipped{a.ArtifactID}); err != nil {
					errors <- errs.Wrap(err, "Could not handle ArtifactDownloadSkipped event: %v", errs.JoinMessage(err))
				}
				if err := s.eventHandler.Handle(events.ArtifactInstallSkipped{a.ArtifactID}); err != nil {
					errors <- errs.Wrap(err, "Could not handle ArtifactInstallSkipped event: %v", errs.JoinMessage(err))
				}
			}
			return
		}

		as, err := s.selectArtifactSetupImplementation(buildResult.BuildEngine, a.ArtifactID)
		if err != nil {
			errors <- errs.Wrap(err, "Failed to select artifact setup implementation")
			return
		}

		unarchiver := as.Unarchiver()
		archivePath, err := s.obtainArtifact(a, unarchiver.Ext())
		if err != nil {
			name := setup.ResolveArtifactName(a.ArtifactID)
			errors <- locale.WrapError(err, "artifact_download_failed", "", name, a.ArtifactID.String())
			return
		}

		err = installFunc(a.ArtifactID, archivePath, as)
		if err != nil {
			name := setup.ResolveArtifactName(a.ArtifactID)
			errors <- locale.WrapError(err, "artifact_setup_failed", "", name, a.ArtifactID.String())
			return
		}
	}
}

func (s *Setup) installFromBuildResult(buildResult *model.BuildResult, artifacts artifact.Map, downloads []artifact.ArtifactDownload, alreadyInstalled store.StoredArtifactMap, setup Setuper, installFunc artifactInstaller) error {
	logging.Debug("Installing artifacts from build result")
	errs, aggregatedErr := aggregateErrors()
	mainthread.Run(func() {
		defer close(errs)
		wp := workerpool.New(MaxConcurrency)
		for _, a := range downloads {
			if _, ok := alreadyInstalled[a.ArtifactID]; ok {
				continue
			}
			var ar *artifact.Artifact
			if arv, ok := artifacts[a.ArtifactID]; ok {
				ar = &arv
			}
			wp.Submit(s.setupArtifactSubmitFunction(a, ar, map[artifact.ArtifactID]struct{}{}, buildResult, setup, installFunc, errs))
		}

		wp.StopWait()
	})

	return <-aggregatedErr
}

func (s *Setup) installFromBuildLog(buildResult *model.BuildResult, artifacts artifact.Map, artifactsToInstall map[artifact.ArtifactID]struct{}, alreadyInstalled store.StoredArtifactMap, setup Setuper, installFunc artifactInstaller, logFilePath string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// The runtime dependencies do not include all build dependencies. Since we are working
	// with the build log, we need to add the missing dependencies to the list of artifacts
	err := buildplan.AddBuildArtifacts(artifacts, buildResult.Build)
	if err != nil {
		return errs.Wrap(err, "Could not add build artifacts to artifact map")
	}

	buildLog, err := buildlog.New(ctx, artifacts, s.eventHandler, buildResult.RecipeID, logFilePath, buildResult)
	if err != nil {
		return errs.Wrap(err, "Cannot establish connection with BuildLog")
	}
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
				var ar *artifact.Artifact
				if arv, ok := artifacts[a.ArtifactID]; ok {
					ar = &arv
				}
				wp.Submit(s.setupArtifactSubmitFunction(a, ar, artifactsToInstall, buildResult, setup, installFunc, errs))
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
	}
	err := fileutils.MoveAllFilesRecursively(
		filepath.Join(unpackedDir, envDef.InstallDir),
		s.store.InstallPath(), onMoveFile,
	)
	if err != nil {
		err := errs.Wrap(err, "Move artifact failed")
		return err
	}

	if err := s.store.StoreArtifact(store.NewStoredArtifact(a, files, dirs, envDef)); err != nil {
		return errs.Wrap(err, "Could not store artifact meta info")
	}

	return nil
}

// downloadArtifact downloads the given artifact
func (s *Setup) downloadArtifact(a artifact.ArtifactDownload, targetFile string) (rerr error) {
	defer func() {
		if rerr != nil {
			if err := s.eventHandler.Handle(events.ArtifactDownloadFailure{a.ArtifactID, rerr}); err != nil {
				rerr = errs.Wrap(rerr, "Could not handle ArtifactDownloadFailure event: %v", errs.JoinMessage(err))
				return
			}
		}
		if err := s.eventHandler.Handle(events.ArtifactDownloadSuccess{a.ArtifactID}); err != nil {
			rerr = errs.Wrap(rerr, "Could not handle ArtifactDownloadSuccess event: %v", errs.JoinMessage(err))
			return
		}
	}()

	artifactURL, err := url.Parse(a.UnsignedURI)
	if err != nil {
		return errs.Wrap(err, "Could not parse artifact URL %s.", a.UnsignedURI)
	}

	b, err := httputil.GetWithProgress(artifactURL.String(), &progress.Report{
		ReportSizeCb: func(size int) error {
			if err := s.eventHandler.Handle(events.ArtifactDownloadStarted{a.ArtifactID, size}); err != nil {
				return errs.Wrap(err, "Could not handle ArtifactDownloadStarted event")
			}
			return nil
		},
		ReportIncrementCb: func(inc int) error {
			if err := s.eventHandler.Handle(events.ArtifactDownloadProgress{a.ArtifactID, inc}); err != nil {
				return errs.Wrap(err, "Could not handle ArtifactDownloadProgress event")
			}
			return nil
		},
	})
	if err != nil {
		return errs.Wrap(err, "Download %s failed", artifactURL.String())
	}
	if err := fileutils.WriteFile(targetFile, b); err != nil {
		return errs.Wrap(err, "Writing download to target file %s failed", targetFile)
	}
	return nil
}

// verifyArtifact verifies the checksum of the downloaded artifact matches the checksum given by the
// platform, and returns an error if the verification fails.
func (s *Setup) verifyArtifact(archivePath string, a artifact.ArtifactDownload) error {
	return validate.Checksum(archivePath, a.Checksum)
}

// obtainArtifact obtains an artifact and returns the local path to that artifact's archive.
func (s *Setup) obtainArtifact(a artifact.ArtifactDownload, extension string) (string, error) {
	if cachedPath, found := s.artifactCache.Get(a.ArtifactID); found {
		if err := s.verifyArtifact(cachedPath, a); err == nil {
			if err := s.eventHandler.Handle(events.ArtifactDownloadSkipped{a.ArtifactID}); err != nil {
				return "", errs.Wrap(err, "Could not handle ArtifactDownloadSkipped event")
			}
			return cachedPath, nil
		}
		// otherwise re-download it; do not return an error
	}

	targetDir := filepath.Join(s.store.InstallPath(), constants.LocalRuntimeTempDirectory)
	if err := fileutils.MkdirUnlessExists(targetDir); err != nil {
		return "", errs.Wrap(err, "Could not create temp runtime dir")
	}

	archivePath := filepath.Join(targetDir, a.ArtifactID.String()+extension)
	if err := s.downloadArtifact(a, archivePath); err != nil {
		return "", errs.Wrap(err, "Could not download artifact %s", a.UnsignedURI)
	}

	err := s.verifyArtifact(archivePath, a)
	if err != nil {
		return "", errs.Wrap(err, "Artifact checksum validation failed")
	}

	err = s.artifactCache.Store(a.ArtifactID, archivePath)
	if err != nil {
		multilog.Error("Could not store artifact in cache: %v", err)
	}

	return archivePath, nil
}

func (s *Setup) unpackArtifact(ua unarchiver.Unarchiver, tarballPath string, targetDir string, progress progress.Reporter) (int, error) {
	f, i, err := ua.PrepareUnpacking(tarballPath, targetDir)
	progress.ReportSize(int(i))
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

func (s *Setup) selectSetupImplementation(buildEngine model.BuildEngine, artifacts artifact.Map) (Setuper, error) {
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

func reusableArtifacts(requestedArtifacts []*bpModel.Artifact, storedArtifacts store.StoredArtifactMap) store.StoredArtifactMap {
	keep := make(store.StoredArtifactMap)

	for _, a := range requestedArtifacts {
		if v, ok := storedArtifacts[a.TargetID]; ok {
			keep[a.TargetID] = v
		}
	}
	return keep
}

func formatBuildPlanError(bperr *model.BuildPlannerError) error {
	var err error = bperr
	// Append last five lines to error message
	offset := 0
	numLines := len(bperr.ValidationErrors())
	if numLines > 5 {
		offset = numLines - 5
	}

	errorLines := strings.Join(bperr.ValidationErrors()[offset:], "\n")
	// Crop at 500 characters to reduce noisy output further
	if len(errorLines) > 500 {
		offset = len(errorLines) - 499
		errorLines = fmt.Sprintf("…%s", errorLines[offset:])
	}
	isCropped := offset > 0
	croppedMessage := ""
	if isCropped {
		croppedMessage = locale.Tl("buildplan_err_cropped_intro", "These are the last lines of the error message:")
	}

	err = locale.WrapError(err, "solver_err", "", croppedMessage, errorLines)
	if bperr.IsTransient() {
		err = errs.AddTips(bperr, locale.Tr("transient_solver_tip"))
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
	logging.Debug("Found %d artifacts to install from '%s'", len(artifacts), *artifactsDir)

	installedArtifacts := make([]artifact.ArtifactID, len(artifacts))

	errors, aggregatedErr := aggregateErrors()
	mainthread.Run(func() {
		defer close(errors)

		wp := workerpool.New(MaxConcurrency)

		for i, a := range artifacts {
			// Each artifact is of the form artifactID.tar.gz, so extract the artifactID from the name.
			filename := a.Path()
			basename := filepath.Base(filename)
			extIndex := strings.Index(basename, ".")
			if extIndex == -1 {
				extIndex = len(basename)
			}
			artifactID := artifact.ArtifactID(basename[0:extIndex])
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
