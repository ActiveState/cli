package setup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	rt "runtime"
	"strings"
	"sync"
	"time"

	"github.com/ActiveState/cli/internal/analytics"
	anaConsts "github.com/ActiveState/cli/internal/analytics/constants"
	"github.com/ActiveState/cli/internal/analytics/dimensions"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/graph"
	"github.com/ActiveState/cli/internal/httputil"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/proxyreader"
	"github.com/ActiveState/cli/internal/rollbar"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/runbits/buildscript"
	"github.com/ActiveState/cli/internal/runbits/dependencies"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/authentication"
	"github.com/ActiveState/cli/pkg/platform/model"
	bpModel "github.com/ActiveState/cli/pkg/platform/model/buildplanner"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifactcache"
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
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/faiface/mainthread"
	"github.com/gammazero/workerpool"
	"github.com/go-openapi/strfmt"
)

// MaxConcurrency is maximum number of parallel artifact installations
const MaxConcurrency = 5

// NotInstalledError is an error returned when the runtime is not completely installed yet.
var NotInstalledError = errs.New("Runtime is not completely installed.")

// BuildError designates a recipe build error.
type BuildError struct {
	*locale.LocalizedError
}

// ArtifactDownloadError designates an error downloading an artifact.
type ArtifactDownloadError struct {
	*errs.WrapperError
}

// ArtifactCachedBuildFailed designates an error due to a build for an artifact that failed and has been cached
type ArtifactCachedBuildFailed struct {
	*errs.WrapperError
	Artifact *buildplan.Artifact
}

// ArtifactInstallError designates an error installing a downloaded artifact.
type ArtifactInstallError struct {
	*errs.WrapperError
}

// ArtifactSetupErrors combines all errors that can happen while installing artifacts in parallel
type ArtifactSetupErrors struct {
	errs []error
}

type ExecutorSetupError struct {
	*errs.WrapperError
}

func (a *ArtifactSetupErrors) Error() string {
	var errors []string
	for _, err := range a.errs {
		errors = append(errors, errs.JoinMessage(err))
	}
	return "Not all artifacts could be installed, errors:\n" + strings.Join(errors, "\n")
}

func (a *ArtifactSetupErrors) Unwrap() []error {
	return a.errs
}

// Errors returns the individual error messages collected from all failing artifact installations
func (a *ArtifactSetupErrors) Errors() []error {
	return a.errs
}

// UserError returns a message including all user-facing sub-error messages
func (a *ArtifactSetupErrors) LocalizedError() string {
	var errStrings []string
	for _, err := range a.errs {
		errStrings = append(errStrings, locale.JoinedErrorMessage(err))
	}
	return locale.Tl("setup_artifacts_err", "Not all artifacts could be installed:\n{{.V0}}", strings.Join(errStrings, "\n"))
}

// ProgressReportError designates an error in the event handler for reporting progress.
type ProgressReportError struct {
	*errs.WrapperError
}

type RuntimeInUseError struct {
	*locale.LocalizedError
	Processes []*graph.ProcessInfo
}

type Targeter interface {
	CommitUUID() strfmt.UUID
	Name() string
	Owner() string
	Dir() string
	Trigger() target.Trigger
	ProjectDir() string

	// ReadOnly communicates that this target should only use cached runtime information (ie. don't check for updates)
	ReadOnly() bool
	// InstallFromDir communicates that this target should only install artifacts from the given directory (i.e. offline installer)
	InstallFromDir() *string
}

type Configurable interface {
	GetString(key string) string
	GetBool(key string) bool
}

type Setup struct {
	auth          *authentication.Auth
	target        Targeter
	eventHandler  events.Handler
	store         *store.Store
	analytics     analytics.Dispatcher
	artifactCache *artifactcache.ArtifactCache
	cfg           Configurable
	out           output.Outputer
	svcm          *model.SvcModel
}

type Setuper interface {
	// DeleteOutdatedArtifacts deletes outdated artifact as best as it can
	DeleteOutdatedArtifacts(*buildplan.ArtifactChangeset, store.StoredArtifactMap, store.StoredArtifactMap) error
}

// ArtifactSetuper is the interface for an implementation of artifact setup functions
// These need to be specialized for each BuildEngine type
type ArtifactSetuper interface {
	EnvDef(tmpInstallDir string) (*envdef.EnvironmentDefinition, error)
	Unarchiver() unarchiver.Unarchiver
}

type ArtifactResolver interface {
	ResolveArtifactName(strfmt.UUID) string
}

type artifactInstaller func(strfmt.UUID, string, ArtifactSetuper) error
type artifactUninstaller func() error

// New returns a new Setup instance that can install a Runtime locally on the machine.
func New(target Targeter, eventHandler events.Handler, auth *authentication.Auth, an analytics.Dispatcher, cfg Configurable, out output.Outputer, svcm *model.SvcModel) *Setup {
	cache, err := artifactcache.New()
	if err != nil {
		multilog.Error("Could not create artifact cache: %v", err)
	}
	return &Setup{auth, target, eventHandler, store.New(target.Dir()), an, cache, cfg, out, svcm}
}

func (s *Setup) Solve() (*bpModel.Commit, error) {
	defer func() {
		s.solveUpdateRecover(recover())
	}()

	if s.target.InstallFromDir() != nil {
		return nil, nil
	}

	if err := s.handleEvent(events.SolveStart{}); err != nil {
		return nil, errs.Wrap(err, "Could not handle SolveStart event")
	}

	bpm := bpModel.NewBuildPlannerModel(s.auth)
	commit, err := bpm.FetchCommit(s.target.CommitUUID(), s.target.Owner(), s.target.Name(), nil)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to fetch build result")
	}

	if err := s.eventHandler.Handle(events.SolveSuccess{}); err != nil {
		return nil, errs.Wrap(err, "Could not handle SolveSuccess event")
	}

	return commit, nil
}

func (s *Setup) Update(commit *bpModel.Commit) (rerr error) {
	defer func() {
		s.solveUpdateRecover(recover())
	}()
	defer func() {
		var ev events.Eventer = events.Success{}
		if rerr != nil {
			ev = events.Failure{}
		}

		err := s.handleEvent(ev)
		if err != nil {
			multilog.Error("Could not handle Success/Failure event: %s", errs.JoinMessage(err))
		}
	}()

	bp := commit.BuildPlan()

	// Do not allow users to deploy runtimes to the root directory (this can easily happen in docker
	// images). Note that runtime targets are fully resolved via fileutils.ResolveUniquePath(), so
	// paths like "/." and "/opt/.." resolve to simply "/" at this time.
	if rt.GOOS != "windows" && s.target.Dir() == "/" {
		return locale.NewInputError("err_runtime_setup_root", "Cannot set up a runtime in the root directory. Please specify or run from a user-writable directory.")
	}

	// Determine if this runtime is currently in use.
	ctx, cancel := context.WithTimeout(context.Background(), model.SvcTimeoutMinimal)
	defer cancel()
	if procs, err := s.svcm.GetProcessesInUse(ctx, ExecDir(s.target.Dir())); err == nil {
		if len(procs) > 0 {
			list := []string{}
			for _, proc := range procs {
				list = append(list, fmt.Sprintf("   - %s (process: %d)", proc.Exe, proc.Pid))
			}
			return &RuntimeInUseError{locale.NewInputError("runtime_setup_in_use_err", "", strings.Join(list, "\n")), procs}
		}
	} else {
		multilog.Error("Unable to determine if runtime is in use: %v", errs.JoinMessage(err))
	}

	// Update all the runtime artifacts
	artifacts, err := s.updateArtifacts(bp)
	if err != nil {
		return errs.Wrap(err, "Failed to update artifacts")
	}

	if err := s.store.StoreBuildPlan(bp); err != nil {
		return errs.Wrap(err, "Could not save recipe file.")
	}

	if err := s.store.StoreBuildScript(commit.BuildScript()); err != nil {
		return errs.Wrap(err, "Could not store buildscript file.")
	}

	if s.target.ProjectDir() != "" && s.cfg.GetBool(constants.OptinBuildscriptsConfig) {
		if err := buildscript_runbit.Update(s.target, commit.BuildScript()); err != nil {
			return errs.Wrap(err, "Could not update build script")
		}
	}

	// Update executors
	if err := s.updateExecutors(artifacts); err != nil {
		return ExecutorSetupError{errs.Wrap(err, "Failed to update executors")}
	}

	// Mark installation as completed
	if err := s.store.MarkInstallationComplete(s.target.CommitUUID(), fmt.Sprintf("%s/%s", s.target.Owner(), s.target.Name())); err != nil {
		return errs.Wrap(err, "Could not mark install as complete.")
	}

	return nil
}

// Panics are serious, and reproducing them in the runtime package is HARD. To help with this we dump
// the build plan when a panic occurs so we have something more to go on.
func (s *Setup) solveUpdateRecover(r interface{}) {
	if r == nil {
		return
	}

	multilog.Critical("Panic during runtime update: %s", r)
	panic(r) // We're just logging the panic while we have context, we're not meant to handle it here
}

func (s *Setup) updateArtifacts(bp *buildplan.BuildPlan) ([]strfmt.UUID, error) {
	mutex := &sync.Mutex{}
	var installArtifactFuncs []func() error

	// Fetch and install each runtime artifact.
	// Note: despite the name, we are "pre-installing" the artifacts to a temporary location.
	// Once all artifacts are fetched, unpacked, and prepared, final installation occurs.
	artifacts, uninstallFunc, err := s.fetchAndInstallArtifacts(bp, func(a strfmt.UUID, archivePath string, as ArtifactSetuper) (rerr error) {
		defer func() {
			if rerr != nil {
				rerr = &ArtifactInstallError{errs.Wrap(rerr, "Unable to install artifact")}
				if err := s.handleEvent(events.ArtifactInstallFailure{a, rerr}); err != nil {
					rerr = errs.Wrap(rerr, "Could not handle ArtifactInstallFailure event")
					return
				}
			}
			if err := s.handleEvent(events.ArtifactInstallSuccess{a}); err != nil {
				rerr = errs.Wrap(rerr, "Could not handle ArtifactInstallSuccess event")
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
				if err := s.handleEvent(events.ArtifactInstallStarted{a, size}); err != nil {
					return errs.Wrap(err, "Could not handle ArtifactInstallStarted event")
				}
				return nil
			},
			ReportIncrementCb: func(inc int) error {
				if err := s.handleEvent(events.ArtifactInstallProgress{a, inc}); err != nil {
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

		mutex.Lock()
		installArtifactFuncs = append(installArtifactFuncs, func() error {
			return s.moveToInstallPath(a, unpackedDir, envDef, numFiles)
		})
		mutex.Unlock()

		return nil
	})
	if err != nil {
		return artifacts, locale.WrapError(err, "err_runtime_setup")
	}

	if os.Getenv(constants.RuntimeSetupWaitEnvVarName) != "" && (condition.OnCI() || condition.BuiltOnDevMachine()) {
		// This code block is for integration testing purposes only.
		// Under normal conditions, we should never access fmt or os.Stdin from this context.
		fmt.Printf("Waiting for input because %s was set\n", constants.RuntimeSetupWaitEnvVarName)
		ch := make([]byte, 1)
		_, err = os.Stdin.Read(ch) // block until input is sent
		if err != nil {
			return artifacts, locale.WrapError(err, "err_runtime_setup")
		}
	}

	// Uninstall outdated artifacts.
	// This must come before calling any installArtifactFuncs or else the runtime may become corrupt.
	if uninstallFunc != nil {
		err := uninstallFunc()
		if err != nil {
			return artifacts, locale.WrapError(err, "err_runtime_setup")
		}
	}

	// Move files to final installation path after successful download and unpack.
	for _, f := range installArtifactFuncs {
		err := f()
		if err != nil {
			return artifacts, locale.WrapError(err, "err_runtime_setup")
		}
	}

	// Clean up temp directory.
	tempDir := filepath.Join(s.store.InstallPath(), constants.LocalRuntimeTempDirectory)
	err = os.RemoveAll(tempDir)
	if err != nil {
		multilog.Log(logging.ErrorNoStacktrace, rollbar.Error)("Failed to remove temporary installation directory %s: %v", tempDir, err)
	}

	return artifacts, nil
}

func (s *Setup) updateExecutors(artifacts []strfmt.UUID) error {
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
// It may also return an artifact uninstaller function that should be run prior to final
// installation.
func (s *Setup) fetchAndInstallArtifacts(bp *buildplan.BuildPlan, installFunc artifactInstaller) ([]strfmt.UUID, artifactUninstaller, error) {
	if s.target.InstallFromDir() != nil {
		artifacts, err := s.fetchAndInstallArtifactsFromDir(installFunc)
		return artifacts, nil, err
	}
	return s.fetchAndInstallArtifactsFromBuildPlan(bp, installFunc)
}

func (s *Setup) fetchAndInstallArtifactsFromBuildPlan(bp *buildplan.BuildPlan, installFunc artifactInstaller) ([]strfmt.UUID, artifactUninstaller, error) {
	// If the build is not ready or if we are installing the buildtime closure
	// then we need to include the buildtime closure in the changed artifacts
	// and the progress reporting.
	includeBuildtimeClosure := strings.EqualFold(os.Getenv(constants.InstallBuildDependencies), "true") || !bp.IsBuildReady()

	platformID, err := model.FilterCurrentPlatform(sysinfo.OS().String(), bp.Platforms(), s.cfg)
	if err != nil {
		return nil, nil, locale.WrapError(err, "err_filter_current_platform")
	}

	artifactFilters := []buildplan.FilterArtifact{
		buildplan.FilterStateArtifacts(),
		buildplan.FilterPlatformArtifacts(platformID),
	}

	// Compute and handle the change summary
	allArtifacts := bp.Artifacts(artifactFilters...)

	// Detect failed artifacts early
	for _, a := range allArtifacts {
		var aErr error
		if a.Status == types.ArtifactFailedPermanently || a.Status == types.ArtifactFailedTransiently {
			errV := &ArtifactCachedBuildFailed{errs.New("artifact failed, status: %s", a.Status), a}
			if aErr == nil {
				aErr = errV
			} else {
				aErr = errs.Pack(aErr, errV)
			}
		}
		if aErr != nil {
			return nil, nil, aErr
		}
	}

	if len(allArtifacts) == 0 {
		v, err := json.Marshal(bp.Artifacts())
		if err != nil {
			return nil, nil, err
		}
		return nil, nil, errs.New("did not find any artifacts that match our platform (%s), full artifacts list: %s", platformID, v)
	}

	resolver, err := selectArtifactResolver(bp)
	if err != nil {
		return nil, nil, errs.Wrap(err, "Failed to select artifact resolver")
	}

	// build results don't have namespace info and will happily report internal only artifacts
	downloadablePrebuiltArtifacts := sliceutils.Filter(allArtifacts, func(a *buildplan.Artifact) bool {
		return a.Status == types.ArtifactSucceeded && a.URL != ""
	})

	// Analytics data to send.
	dimensions := &dimensions.Values{
		CommitID: ptr.To(s.target.CommitUUID().String()),
	}

	// send analytics build event, if a new runtime has to be built in the cloud
	if bp.IsBuildInProgress() {
		s.analytics.Event(anaConsts.CatRuntimeDebug, anaConsts.ActRuntimeBuild, dimensions)
	}

	oldBuildPlan, err := s.store.BuildPlan()
	if err != nil && !errors.As(err, ptr.To(&store.ErrVersionMarker{})) {
		return nil, nil, errs.Wrap(err, "could not load existing build plan")
	}

	var oldBuildPlanArtifacts buildplan.Artifacts
	var changedArtifacts *buildplan.ArtifactChangeset
	if oldBuildPlan != nil {
		oldBuildPlanArtifacts = oldBuildPlan.Artifacts(artifactFilters...)
		changedArtifacts = ptr.To(bp.DiffArtifacts(oldBuildPlan, true))
	}

	storedArtifacts, err := s.store.Artifacts()
	if err != nil {
		return nil, nil, locale.WrapError(err, "err_stored_artifacts")
	}

	alreadyInstalled := reusableArtifacts(allArtifacts, storedArtifacts)

	artifactNamesList := []string{}
	for _, a := range allArtifacts {
		artifactNamesList = append(artifactNamesList, a.Name())
	}
	installedList := []string{}
	for _, a := range alreadyInstalled {
		installedList = append(installedList, resolver.ResolveArtifactName(a.ArtifactID))
	}
	downloadList := []string{}
	for _, a := range downloadablePrebuiltArtifacts {
		downloadList = append(downloadList, resolver.ResolveArtifactName(a.ArtifactID))
	}
	logging.Debug(
		"Parsed artifacts.\nBuild ready: %v\nArtifact names: %v\nAlready installed: %v\nTo Download: %v",
		bp.IsBuildReady(), artifactNamesList, installedList, downloadList,
	)

	filterNeedsInstall := func(a *buildplan.Artifact) bool {
		_, alreadyInstalled := alreadyInstalled[a.ArtifactID]
		return !alreadyInstalled
	}
	filters := []buildplan.FilterArtifact{filterNeedsInstall}
	if !includeBuildtimeClosure {
		filters = append(filters, buildplan.FilterRuntimeArtifacts())
	}
	artifactsToInstall := allArtifacts.Filter(filters...)
	if err != nil {
		return nil, nil, errs.Wrap(err, "Failed to compute artifacts to build")
	}

	// Output a dependency summary if applicable.
	if s.target.Trigger() == target.TriggerCheckout {
		dependencies.OutputSummary(s.out, bp.RequestedArtifacts())
	} else if s.target.Trigger() == target.TriggerInit {
		artifacts := bp.Artifacts().Filter(buildplan.FilterStateArtifacts(), buildplan.FilterRuntimeArtifacts())
		dependencies.OutputSummary(s.out, artifacts)
	} else if len(oldBuildPlanArtifacts) > 0 && changedArtifacts != nil {
		dependencies.OutputChangeSummary(s.out, changedArtifacts, oldBuildPlanArtifacts)
	}

	// The log file we want to use for builds
	logFilePath := logging.FilePathFor(fmt.Sprintf("build-%s.log", s.target.CommitUUID().String()+"-"+time.Now().Format("20060102150405")))

	recipeID, err := bp.RecipeID()
	if err != nil {
		return nil, nil, errs.Wrap(err, "Could not get recipe ID from build plan")
	}

	artifactNameMap := map[strfmt.UUID]string{}
	for _, a := range allArtifacts {
		artifactNameMap[a.ArtifactID] = a.Name()
	}

	if err := s.eventHandler.Handle(events.Start{
		RecipeID:         recipeID,
		RequiresBuild:    bp.IsBuildInProgress(),
		Artifacts:        artifactNameMap,
		LogFilePath:      logFilePath,
		ArtifactsToBuild: allArtifacts.ToIDSlice(),
		// Yes these have the same value; this is intentional.
		// Separating these out just allows us to be more explicit and intentional in our event handling logic.
		ArtifactsToDownload: artifactsToInstall.ToIDSlice(),
		ArtifactsToInstall:  artifactsToInstall.ToIDSlice(),
	}); err != nil {
		return nil, nil, errs.Wrap(err, "Could not handle Start event")
	}

	var uninstallArtifacts artifactUninstaller = func() error {
		setup, err := s.selectSetupImplementation(bp.Engine())
		if err != nil {
			return errs.Wrap(err, "Failed to select setup implementation")
		}
		return s.deleteOutdatedArtifacts(setup, changedArtifacts, alreadyInstalled)
	}

	// only send the download analytics event, if we have to install artifacts that are not yet installed
	if len(artifactsToInstall) > 0 {
		// if we get here, we download artifacts
		s.analytics.Event(anaConsts.CatRuntimeDebug, anaConsts.ActRuntimeDownload, dimensions)
	}

	err = s.installArtifactsFromBuild(bp.IsBuildReady(), bp.Engine(), recipeID, artifactsToInstall, installFunc, logFilePath)
	if err != nil {
		return nil, nil, err
	}
	err = s.artifactCache.Save()
	if err != nil {
		multilog.Error("Could not save artifact cache updates: %v", err)
	}

	artifactIDs := allArtifacts.ToIDSlice()
	logging.Debug("Returning artifacts: %v", artifactIDs)
	return artifactIDs, uninstallArtifacts, nil
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

func (s *Setup) installArtifactsFromBuild(isReady bool, engine types.BuildEngine, recipeID strfmt.UUID, artifacts buildplan.Artifacts, installFunc artifactInstaller, logFilePath string) error {
	// Artifacts are installed in two stages
	// - The first stage runs concurrently in MaxConcurrency worker threads (download, unpacking, relocation)
	// - The second stage moves all files into its final destination is running in a single thread (using the mainthread library) to avoid file conflicts

	var err error
	if isReady {
		logging.Debug("Installing via build result")
		if err := s.handleEvent(events.BuildSkipped{}); err != nil {
			return errs.Wrap(err, "Could not handle BuildSkipped event")
		}
		err = s.installFromBuildResult(engine, artifacts, installFunc)
		if err != nil {
			err = errs.Wrap(err, "Installing via build result failed")
		}
	} else {
		logging.Debug("Installing via buildlog streamer")
		err = s.installFromBuildLog(engine, recipeID, artifacts, installFunc, logFilePath)
		if err != nil {
			err = errs.Wrap(err, "Installing via buildlog streamer failed")
		}
	}

	return err
}

// setupArtifactSubmitFunction returns a function that sets up an artifact and can be submitted to a workerpool
func (s *Setup) setupArtifactSubmitFunction(
	engine types.BuildEngine,
	ar *buildplan.Artifact,
	installFunc artifactInstaller,
	errors chan<- error,
) func() {
	return func() {
		as, err := s.selectArtifactSetupImplementation(engine, ar.ArtifactID)
		if err != nil {
			errors <- errs.Wrap(err, "Failed to select artifact setup implementation")
			return
		}

		unarchiver := as.Unarchiver()
		archivePath, err := s.obtainArtifact(ar, unarchiver.Ext())
		if err != nil {
			errors <- locale.WrapError(err, "artifact_download_failed", "", ar.Name(), ar.ArtifactID.String())
			return
		}

		err = installFunc(ar.ArtifactID, archivePath, as)
		if err != nil {
			errors <- locale.WrapError(err, "artifact_setup_failed", "", ar.Name(), ar.ArtifactID.String())
			return
		}
	}
}

func (s *Setup) installFromBuildResult(engine types.BuildEngine, artifacts buildplan.Artifacts, installFunc artifactInstaller) error {
	logging.Debug("Installing artifacts from build result")
	errs, aggregatedErr := aggregateErrors()
	mainthread.Run(func() {
		defer close(errs)
		wp := workerpool.New(MaxConcurrency)
		for _, a := range artifacts {
			wp.Submit(s.setupArtifactSubmitFunction(engine, a, installFunc, errs))
		}

		wp.StopWait()
	})

	return <-aggregatedErr
}

func (s *Setup) installFromBuildLog(engine types.BuildEngine, recipeID strfmt.UUID, artifacts buildplan.Artifacts, installFunc artifactInstaller, logFilePath string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	buildLog, err := buildlog.New(ctx, artifacts.ToIDMap(), s.eventHandler, recipeID, logFilePath)
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
				wp.Submit(s.setupArtifactSubmitFunction(engine, a, installFunc, errs))
			}
		}()

		if err = buildLog.Wait(); err != nil {
			errs <- err
		}
	})

	return <-aggregatedErr
}

func (s *Setup) moveToInstallPath(a strfmt.UUID, unpackedDir string, envDef *envdef.EnvironmentDefinition, numFiles int) error {
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
func (s *Setup) downloadArtifact(a *buildplan.Artifact, targetFile string) (rerr error) {
	defer func() {
		if rerr != nil {
			if !errs.Matches(rerr, &ProgressReportError{}) {
				rerr = &ArtifactDownloadError{errs.Wrap(rerr, "Unable to download artifact")}
			}

			if err := s.handleEvent(events.ArtifactDownloadFailure{a.ArtifactID, rerr}); err != nil {
				rerr = errs.Wrap(rerr, "Could not handle ArtifactDownloadFailure event")
				return
			}
		}

		if err := s.handleEvent(events.ArtifactDownloadSuccess{a.ArtifactID}); err != nil {
			rerr = errs.Wrap(rerr, "Could not handle ArtifactDownloadSuccess event")
			return
		}
	}()

	if a.URL == "" {
		return errs.New("Artifact URL is empty: %+v", a)
	}

	artifactURL, err := url.Parse(a.URL)
	if err != nil {
		return errs.Wrap(err, "Could not parse artifact URL %s.", a.URL)
	}

	b, err := httputil.GetWithProgress(artifactURL.String(), &progress.Report{
		ReportSizeCb: func(size int) error {
			if err := s.handleEvent(events.ArtifactDownloadStarted{a.ArtifactID, size}); err != nil {
				return ProgressReportError{errs.Wrap(err, "Could not handle ArtifactDownloadStarted event")}
			}
			return nil
		},
		ReportIncrementCb: func(inc int) error {
			if err := s.handleEvent(events.ArtifactDownloadProgress{a.ArtifactID, inc}); err != nil {
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
func (s *Setup) verifyArtifact(archivePath string, a *buildplan.Artifact) error {
	return validate.Checksum(archivePath, a.Checksum)
}

// obtainArtifact obtains an artifact and returns the local path to that artifact's archive.
func (s *Setup) obtainArtifact(a *buildplan.Artifact, extension string) (string, error) {
	if cachedPath, found := s.artifactCache.Get(a.ArtifactID); found {
		if err := s.verifyArtifact(cachedPath, a); err == nil {
			if err := s.handleEvent(events.ArtifactDownloadSkipped{a.ArtifactID}); err != nil {
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
		return "", errs.Wrap(err, "Could not download artifact %s", a.URL)
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
	if err != nil {
		return 0, errs.Wrap(err, "Prepare for unpacking failed")
	}
	defer f.Close()

	if err := progress.ReportSize(int(i)); err != nil {
		return 0, errs.Wrap(err, "Could not report size")
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

func (s *Setup) selectSetupImplementation(buildEngine types.BuildEngine) (Setuper, error) {
	switch buildEngine {
	case types.Alternative:
		return alternative.NewSetup(s.store), nil
	case types.Camel:
		return camel.NewSetup(s.store), nil
	default:
		return nil, errs.New("Unknown build engine: %s", buildEngine)
	}
}

func selectArtifactResolver(bp *buildplan.BuildPlan) (ArtifactResolver, error) {
	switch bp.Engine() {
	case types.Alternative:
		return alternative.NewResolver(bp.Artifacts().ToIDMap()), nil
	case types.Camel:
		return camel.NewResolver(), nil
	default:
		return nil, errs.New("Unknown build engine: %s", bp.Engine())
	}
}

func (s *Setup) selectArtifactSetupImplementation(buildEngine types.BuildEngine, a strfmt.UUID) (ArtifactSetuper, error) {
	switch buildEngine {
	case types.Alternative:
		return alternative.NewArtifactSetup(a, s.store), nil
	case types.Camel:
		return camel.NewArtifactSetup(a, s.store), nil
	default:
		return nil, errs.New("Unknown build engine: %s", buildEngine)
	}
}

func ExecDir(targetDir string) string {
	return filepath.Join(targetDir, "exec")
}

func reusableArtifacts(requestedArtifacts []*buildplan.Artifact, storedArtifacts store.StoredArtifactMap) store.StoredArtifactMap {
	keep := make(store.StoredArtifactMap)

	for _, a := range requestedArtifacts {
		if v, ok := storedArtifacts[a.ArtifactID]; ok {
			keep[a.ArtifactID] = v
		}
	}
	return keep
}

func (s *Setup) fetchAndInstallArtifactsFromDir(installFunc artifactInstaller) ([]strfmt.UUID, error) {
	artifactsDir := s.target.InstallFromDir()
	if artifactsDir == nil {
		return nil, errs.New("Cannot install from a directory that is nil")
	}

	artifacts, err := fileutils.ListDir(*artifactsDir, false)
	if err != nil {
		return nil, errs.Wrap(err, "Cannot read from directory to install from")
	}
	logging.Debug("Found %d artifacts to install from '%s'", len(artifacts), *artifactsDir)

	installedArtifacts := make([]strfmt.UUID, len(artifacts))

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
			artifactID := strfmt.UUID(basename[0:extIndex])
			installedArtifacts[i] = artifactID

			// Submit the artifact for setup and install.
			func(filename string, artifactID strfmt.UUID) {
				wp.Submit(func() {
					as := alternative.NewArtifactSetup(artifactID, s.store) // offline installer artifacts are in this format
					err = installFunc(artifactID, filename, as)
					if err != nil {
						errors <- locale.WrapError(err, "artifact_setup_failed", "", artifactID.String(), "")
					}
				})
			}(filename, artifactID) // avoid referencing loop variables inside goroutine closures
		}

		wp.StopWait()
	})

	return installedArtifacts, <-aggregatedErr
}

func (s *Setup) handleEvent(ev events.Eventer) error {
	err := s.eventHandler.Handle(ev)
	if err != nil {
		return &ProgressReportError{errs.Wrap(err, "Error handling event: %v", errs.JoinMessage(err))}
	}
	return nil
}

func (s *Setup) deleteOutdatedArtifacts(setup Setuper, changedArtifacts *buildplan.ArtifactChangeset, alreadyInstalled store.StoredArtifactMap) error {
	storedArtifacts, err := s.store.Artifacts()
	if err != nil {
		return locale.WrapError(err, "err_stored_artifacts")
	}

	err = setup.DeleteOutdatedArtifacts(changedArtifacts, storedArtifacts, alreadyInstalled)
	if err != nil {
		// This multilog is technically redundant and may be dropped after we can collect data on this error for a while as rollbar is not surfacing the returned error
		// https://github.com/ActiveState/cli/pull/2620#discussion_r1256103647
		multilog.Error("Could not delete outdated artifacts: %s", errs.JoinMessage(err))
		return errs.Wrap(err, "Could not delete outdated artifacts")
	}
	return nil
}
