package runtime

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/pkg/executors"
	"github.com/go-openapi/strfmt"
	"golang.org/x/net/context"

	"github.com/ActiveState/cli/internal/chanutils/workerpool"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/httputil"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/proxyreader"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/platform/api/buildplanner/types"
	"github.com/ActiveState/cli/pkg/platform/model"
	"github.com/ActiveState/cli/pkg/runtime/events"
	"github.com/ActiveState/cli/pkg/runtime/events/progress"
	"github.com/ActiveState/cli/pkg/runtime/internal/buildlog"
	"github.com/ActiveState/cli/pkg/runtime/internal/camel"
	"github.com/ActiveState/cli/pkg/runtime/internal/envdef"
	"github.com/ActiveState/cli/pkg/sysinfo"
)

// maxConcurrency is the maximum number of concurrent workers that can be running at any given time during an update
const maxConcurrency = 5

// fromArchive contains options for setting up a runtime from an archive.
type fromArchive struct {
	Dir         string
	PlatformID  strfmt.UUID
	ArtifactExt string
}

type Opts struct {
	PreferredLibcVersion string
	EventHandlers        []events.HandlerFunc
	BuildlogFilePath     string
	BuildProgressUrl     string
	Portable             bool
	CacheSize            int

	FromArchive *fromArchive

	// Annotations are used strictly to pass information for the purposes of analytics
	// These should never be used for business logic. If the need to use them for business logic arises either we are
	// going down a wrong rabbit hole or we need to revisit the architecture.
	Annotations struct {
		Owner      string
		Project    string
		CommitUUID strfmt.UUID
	}
}

type SetOpt func(*Opts)

type setup struct {
	path              string
	opts              *Opts
	depot             *depot
	ecosystems        []ecosystem
	supportsHardLinks bool
	env               *envdef.Collection
	buildplan         *buildplan.BuildPlan

	// toBuild encompasses all artifacts that will need to be build for this runtime.
	// This does NOT mean every artifact in the runtime closure if this is an update (as oppose to a fresh toInstall).
	// Because when we update we likely already have some of the requisite artifacts installed, and thus we don't need their toBuild.
	toBuild buildplan.ArtifactIDMap

	// toDownload encompasses all artifacts that will need to be downloaded for this runtime. The same caveat applies as for toBuild.
	toDownload buildplan.ArtifactIDMap

	// toUnpack encompasses all artifacts that will need to be unpacked for this runtime.
	// This is identical to toDownload except when setting up a runtime from an archive. In that case,
	// toDownload is nil.
	// The same caveat applies as for toBuild.
	toUnpack buildplan.ArtifactIDMap

	// toInstall encompasses all artifacts that will need to be installed for this runtime. The same caveat applies as for toBuild.
	toInstall buildplan.ArtifactIDMap

	// toUninstall encompasses all artifacts that will need to be uninstalled for this runtime.
	toUninstall map[strfmt.UUID]bool
}

func newSetup(path string, bp *buildplan.BuildPlan, env *envdef.Collection, depot *depot, opts *Opts) (*setup, error) {
	depot.SetCacheSize(opts.CacheSize)
	installedArtifacts := depot.List(path)

	var platformID strfmt.UUID
	if opts.FromArchive == nil {
		var err error
		platformID, err = model.FilterCurrentPlatform(sysinfo.OS().String(), bp.Platforms(), opts.PreferredLibcVersion)
		if err != nil {
			return nil, errs.Wrap(err, "Could not get platform ID")
		}
	} else {
		platformID = opts.FromArchive.PlatformID
	}

	filterInstallable := []buildplan.FilterArtifact{
		buildplan.FilterPlatformArtifacts(platformID),
		buildplan.FilterStateArtifacts(),
	}
	if os.Getenv(constants.InstallBuildDependenciesEnvVarName) != "true" {
		filterInstallable = append(filterInstallable, buildplan.FilterRuntimeArtifacts())
	}

	// Start off with the full range of artifacts relevant to our platform
	installableArtifacts := bp.Artifacts(filterInstallable...)

	// Identify which artifacts we'll need to install, this filters out any artifacts that are already installed.
	artifactsToInstall := installableArtifacts.Filter(func(a *buildplan.Artifact) bool {
		_, installed := installedArtifacts[a.ArtifactID]
		return !installed
	})

	// Identify which artifacts we can uninstall
	installableArtifactsMap := installableArtifacts.ToIDMap()
	artifactsToUninstall := map[strfmt.UUID]bool{}
	for id := range installedArtifacts {
		if _, required := installableArtifactsMap[id]; !required {
			artifactsToUninstall[id] = true
		}
	}

	// Calculate which artifacts need to be downloaded; if an artifact we want to install is not in our depot then
	// by definition we'll need to download it (unless we're setting up the runtime from an archive).
	// We also calculate which artifacts are immediately ready to be installed, as its the inverse condition of the above.
	artifactsToDownload := artifactsToInstall.Filter(func(a *buildplan.Artifact) bool {
		return !depot.Exists(a.ArtifactID)
	})
	artifactsToUnpack := artifactsToDownload
	if opts.FromArchive != nil {
		artifactsToDownload = nil
	}

	// Now that we know which artifacts we'll need to download we can use this as our basis for calculating which artifacts
	// still need to be build. This encompasses the artifacts themselves, as well as any of their dependencies. And of
	// course we only want to filter artifacts that actually require a build, as the build may be cached server side.
	artifactsToBuild := append(artifactsToDownload, artifactsToDownload.Dependencies(true, nil)...).Filter(buildplan.FilterNotBuild())
	artifactsToBuild = sliceutils.UniqueByProperty(artifactsToBuild, func(a *buildplan.Artifact) any { return a.ArtifactID })

	// Check for cached build failures
	for _, a := range artifactsToBuild {
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
			return nil, aErr
		}
	}

	// Load all ecosystems
	var ecosystems []ecosystem
	for _, e := range availableEcosystems {
		ecosystem := e()
		if err := ecosystem.Init(path, bp); err != nil {
			return nil, errs.Wrap(err, "Could not create ecosystem")
		}
		ecosystems = append(ecosystems, ecosystem)
	}

	return &setup{
		path:              path,
		opts:              opts,
		env:               env,
		depot:             depot,
		supportsHardLinks: supportsHardLinks(depot.depotPath),
		buildplan:         bp,
		toBuild:           artifactsToBuild.ToIDMap(),
		toDownload:        artifactsToDownload.ToIDMap(),
		toUnpack:          artifactsToUnpack.ToIDMap(),
		toInstall:         artifactsToInstall.ToIDMap(),
		toUninstall:       artifactsToUninstall,
		ecosystems:        ecosystems,
	}, nil
}

func (s *setup) RunAndWait() (rerr error) {
	defer func() {
		// Handle success / failure event
		var name = "success"
		var ev events.Event = events.Success{}
		if rerr != nil {
			name = "failure"
			ev = events.Failure{
				Error: rerr,
			}
		}

		err := s.fireEvent(ev)
		if err != nil {
			rerr = errs.Pack(rerr, errs.Wrap(err, "Could not handle %s event", name))
		}
	}()

	if err := s.fireEvent(events.Start{
		RecipeID:            s.buildplan.LegacyRecipeID(),
		RequiresBuild:       s.buildplan.IsBuildInProgress() && len(s.toDownload) > 0,
		LogFilePath:         s.opts.BuildlogFilePath,
		ProgressUrl:         s.opts.BuildProgressUrl,
		ArtifactsToBuild:    s.toBuild,
		ArtifactsToDownload: s.toDownload,
		ArtifactsToUnpack:   s.toUnpack,
		ArtifactsToInstall:  s.toInstall,
	}); err != nil {
		return errs.Wrap(err, "Could not handle Start event")
	}

	if err := s.update(); err != nil {
		return errs.Wrap(err, "Could not update")
	}

	return nil
}

func (s *setup) update() error {
	if err := fileutils.MkdirUnlessExists(filepath.Join(s.path, configDir)); err != nil {
		return errs.Wrap(err, "Could not create runtime config dir")
	}

	blog := buildlog.New(s.buildplan.LegacyRecipeID(), s.toBuild).
		WithEventHandler(s.opts.EventHandlers...).
		WithLogFile(filepath.Join(s.path, configDir, buildLogFile))

	// Download artifacts when ready, or unpack artifacts from archive.
	// Note: if there are artifacts to download, s.toUnpack == s.toDownload, and downloaded artifacts
	// are unpacked in the same step.
	wp := workerpool.New(maxConcurrency)
	for _, a := range s.toUnpack { // iterate over unpack as downloads will not be set if installing from archive
		s.onArtifactBuildReady(blog, a, func() {
			wp.Submit(func() error {
				if err := s.obtain(a); err != nil {
					return errs.Wrap(err, "obtain failed")
				}
				return nil
			})
		})
	}

	// Wait for build to finish
	if !s.buildplan.IsBuildReady() && len(s.toBuild) > 0 {
		if err := blog.Wait(context.Background()); err != nil {
			return errs.Wrap(err, "errors occurred during buildlog streaming")
		}
	}

	// Wait for workerpool handling build results to finish
	if err := wp.Wait(); err != nil {
		return errs.Wrap(err, "errors occurred during obtain")
	}

	// Now we start modifying the runtime directory
	// This happens AFTER all the download steps are finished, and should be very fast because installing is mostly just
	// creating links to the depot.
	// We do this as a separate step so we don't leave the runtime dir in a half-installed state if issues happen earlier
	// on in the process.

	// Uninstall artifacts
	for id := range s.toUninstall {
		if err := s.uninstall(id); err != nil {
			return errs.Wrap(err, "Could not uninstall artifact")
		}
	}

	// Install artifacts
	wp = workerpool.New(maxConcurrency)
	for _, a := range s.toInstall {
		wp.Submit(func() error {
			if err := s.install(a); err != nil {
				return errs.Wrap(err, "Could not install artifact")
			}
			return nil
		})
	}

	// Wait for workerpool handling artifact installs to finish
	if err := wp.Wait(); err != nil {
		return errs.Wrap(err, "errors occurred during install")
	}

	if err := s.postProcess(); err != nil {
		return errs.Wrap(err, "Postprocessing failed")
	}

	return nil
}

func (s *setup) onArtifactBuildReady(blog *buildlog.BuildLog, artifact *buildplan.Artifact, cb func()) {
	if _, ok := s.toBuild[artifact.ArtifactID]; !ok {
		// No need to build, artifact can already be downloaded
		cb()
		return
	}

	blog.OnArtifactReady(artifact.ArtifactID, cb)
}

func (s *setup) obtain(artifact *buildplan.Artifact) (rerr error) {
	var b []byte
	if s.opts.FromArchive == nil {
		// Download artifact
		var err error
		b, err = s.download(artifact)
		if err != nil {
			return errs.Wrap(err, "download failed")
		}
	} else {
		// Read the artifact from the archive.
		var err error
		name := artifact.ArtifactID.String() + s.opts.FromArchive.ArtifactExt
		artifactFile := filepath.Join(s.opts.FromArchive.Dir, name)
		b, err = fileutils.ReadFile(artifactFile)
		if err != nil {
			return errs.Wrap(err, "read from archive failed")
		}
	}

	// Unpack artifact
	if err := s.unpack(artifact, b); err != nil {
		return errs.Wrap(err, "unpack failed")
	}

	return nil
}

func (s *setup) download(artifact *buildplan.Artifact) (_ []byte, rerr error) {
	defer func() {
		if rerr != nil {
			if err := s.fireEvent(events.ArtifactDownloadFailure{artifact.ArtifactID, rerr}); err != nil {
				rerr = errs.Pack(rerr, errs.Wrap(err, "Could not handle ArtifactDownloadFailure event"))
			}
		}
	}()

	b, err := httputil.GetWithProgress(artifact.URL, &progress.Report{
		ReportSizeCb: func(size int) error {
			if err := s.fireEvent(events.ArtifactDownloadStarted{artifact.ArtifactID, size}); err != nil {
				return ProgressReportError{errs.Wrap(err, "Could not handle ArtifactDownloadStarted event")}
			}
			return nil
		},
		ReportIncrementCb: func(inc int) error {
			if err := s.fireEvent(events.ArtifactDownloadProgress{artifact.ArtifactID, inc}); err != nil {
				return errs.Wrap(err, "Could not handle ArtifactDownloadProgress event")
			}
			return nil
		},
	})
	if err != nil {
		return nil, errs.Wrap(err, "Download %s failed", artifact.URL)
	}
	if err := s.fireEvent(events.ArtifactDownloadSuccess{artifact.ArtifactID}); err != nil {
		return nil, errs.Wrap(errs.Pack(err, err), "Could not handle ArtifactDownloadSuccess event")
	}

	return b, nil
}

func (s *setup) unpack(artifact *buildplan.Artifact, b []byte) (rerr error) {
	defer func() {
		if rerr != nil {
			if err := s.fireEvent(events.ArtifactUnpackFailure{artifact.ArtifactID, rerr}); err != nil {
				rerr = errs.Pack(rerr, errs.Wrap(err, "Could not handle ArtifactUnpackFailure event"))
			}
		} else {
			if err := s.fireEvent(events.ArtifactUnpackSuccess{artifact.ArtifactID}); err != nil {
				rerr = errs.Pack(rerr, errs.Wrap(errs.Pack(err, err), "Could not handle ArtifactUnpackSuccess event"))
			}
		}
	}()

	var ua unarchiver.Unarchiver = unarchiver.NewTarGz()
	if strings.HasSuffix(strings.ToLower(artifact.URL), "zip") {
		ua = unarchiver.NewZip()
	}

	if err := s.fireEvent(events.ArtifactUnpackStarted{artifact.ArtifactID, len(b)}); err != nil {
		return errs.Wrap(err, "Could not handle ArtifactUnpackStarted event")
	}

	proxy := proxyreader.NewProxyReader(&progress.Report{
		ReportIncrementCb: func(inc int) error {
			if err := s.fireEvent(events.ArtifactUnpackProgress{artifact.ArtifactID, inc}); err != nil {
				return errs.Wrap(err, "Could not handle ArtifactUnpackProgress event")
			}
			return nil
		},
	}, bytes.NewReader(b))
	if err := ua.Unarchive(proxy, int64(len(b)), s.depot.Path(artifact.ArtifactID)); err != nil {
		return errs.Wrap(err, "unpack failed")
	}

	if err := s.depot.Put(artifact.ArtifactID); err != nil {
		return errs.Wrap(err, "Could not put artifact in depot")
	}

	// Camel artifacts do not have runtime.json, so in order to not have multiple paths of logic we generate one based
	// on the camel specific info in the artifact.
	if s.buildplan.Engine() == types.Camel {
		artifactDepotPath := s.depot.Path(artifact.ArtifactID)
		envDef, err := camel.NewEnvironmentDefinitions(artifactDepotPath)
		if err != nil {
			return errs.Wrap(err, "Could not get camel env")
		}

		if err := envDef.Save(artifactDepotPath); err != nil {
			return errs.Wrap(err, "Could not save camel env")
		}
	}

	return nil
}

func (s *setup) updateExecutors() error {
	execPath := ExecutorsPath(s.path)
	if err := fileutils.MkdirUnlessExists(execPath); err != nil {
		return errs.Wrap(err, "Could not create executors directory")
	}

	env, err := s.env.Environment(s.path, false)
	if err != nil {
		return errs.Wrap(err, "Could not get env")
	}

	exePaths, err := osutils.ExecutablePaths(env)
	if err != nil {
		return errs.Wrap(err, "Could not get executable paths")
	}

	execInit := executors.New(execPath)
	if err := execInit.Apply(svcctl.NewIPCSockPathFromGlobals().String(), executors.NewTarget(
		s.opts.Annotations.CommitUUID,
		s.opts.Annotations.Owner,
		s.opts.Annotations.Project,
		s.path,
	), env, exePaths); err != nil {
		return locale.WrapError(err, "err_deploy_executors", "Could not create executors")
	}

	return nil
}

func (s *setup) install(artifact *buildplan.Artifact) (rerr error) {
	id := artifact.ArtifactID
	defer func() {
		if rerr == nil {
			if err := s.fireEvent(events.ArtifactInstallSuccess{id}); err != nil {
				rerr = errs.Pack(rerr, errs.Wrap(err, "Could not handle ArtifactInstallSuccess event"))
			}
		} else {
			if err := s.fireEvent(events.ArtifactInstallFailure{id, rerr}); err != nil {
				rerr = errs.Pack(rerr, errs.Wrap(err, "Could not handle ArtifactInstallFailure event"))
			}
		}
	}()

	if err := s.fireEvent(events.ArtifactInstallStarted{id}); err != nil {
		return errs.Wrap(err, "Could not handle ArtifactInstallStarted event")
	}

	artifactDepotPath := s.depot.Path(id)

	if ecosys := filterEcosystemMatchingArtifact(artifact, s.ecosystems); ecosys != nil {
		files, err := ecosys.Add(artifact, artifactDepotPath)
		if err != nil {
			return errs.Wrap(err, "Ecosystem unable to add artifact")
		}
		s.depot.Track(id, &deployment{
			Type:  deploymentTypeEcosystem,
			Path:  s.path,
			Files: files,
		})
		return nil
	}

	envDef, err := s.env.Load(artifactDepotPath)
	if err != nil {
		return errs.Wrap(err, "Could not get env")
	}

	if envDef.NeedsTransforms() || !s.supportsHardLinks || s.opts.Portable {
		if err := s.depot.DeployViaCopy(id, envDef.InstallDir, s.path); err != nil {
			return errs.Wrap(err, "Could not deploy artifact via copy")
		}
		if envDef.NeedsTransforms() {
			if err := envDef.ApplyFileTransforms(s.path); err != nil {
				return errs.Wrap(err, "Could not apply env transforms")
			}
		}
	} else {
		if err := s.depot.DeployViaLink(id, envDef.InstallDir, s.path); err != nil {
			return errs.Wrap(err, "Could not deploy artifact via link")
		}
	}

	return nil
}

func (s *setup) uninstall(id strfmt.UUID) (rerr error) {
	defer func() {
		if rerr == nil {
			if err := s.fireEvent(events.ArtifactUninstallSuccess{id}); err != nil {
				rerr = errs.Pack(rerr, errs.Wrap(err, "Could not handle ArtifactUninstallSuccess event"))
			}
		} else {
			if err := s.fireEvent(events.ArtifactUninstallFailure{id, rerr}); err != nil {
				rerr = errs.Pack(rerr, errs.Wrap(err, "Could not handle ArtifactUninstallFailure event"))
			}
		}
	}()

	if err := s.fireEvent(events.ArtifactUninstallStarted{id}); err != nil {
		return errs.Wrap(err, "Could not handle ArtifactUninstallStarted event")
	}

	artifactDepotPath := s.depot.Path(id)

	// TODO: CP-956
	//if ecosys := filterEcosystemMatchingArtifact(artifact, s.ecosystems); ecosys != nil {
	//	err := ecosys.Remove(artifact)
	//	if err != nil {
	//		return errs.Wrap(err, "Ecosystem unable to remove artifact")
	//	}
	//	s.depot.Untrack(id, filepath.Join(s.path, artifact.ArtifactID.String()))
	//	return nil
	//}

	envDef, err := s.env.Load(artifactDepotPath)
	if err != nil {
		return errs.Wrap(err, "Could not get env")
	}

	if err := s.env.Unload(artifactDepotPath); err != nil {
		return errs.Wrap(err, "Could not unload artifact envdef")
	}

	if err := s.depot.Undeploy(id, envDef.InstallDir, s.path); err != nil {
		return errs.Wrap(err, "Could not unlink artifact")
	}

	return nil
}

func (s *setup) postProcess() (rerr error) {
	if err := s.fireEvent(events.PostProcessStarted{}); err != nil {
		return errs.Wrap(err, "Could not handle PostProcessStarted event")
	}

	defer func() {
		if rerr == nil {
			if err := s.fireEvent(events.PostProcessSuccess{}); err != nil {
				rerr = errs.Pack(rerr, errs.Wrap(err, "Could not handle PostProcessSuccess event"))
			}
		} else {
			if err := s.fireEvent(events.PostProcessFailure{rerr}); err != nil {
				rerr = errs.Pack(rerr, errs.Wrap(err, "Could not handle PostProcessFailure event"))
			}
		}
	}()

	// Tell applicable ecosystems to apply changes.
	for _, e := range s.ecosystems {
		if err := e.Apply(); err != nil {
			return errs.Wrap(err, "Could not apply ecosystem changes")
		}
	}

	// Update executors
	if err := s.updateExecutors(); err != nil {
		return errs.Wrap(err, "Could not update executors")
	}

	// Save depot changes
	if err := s.depot.Save(); err != nil {
		return errs.Wrap(err, "Could not save depot")
	}

	return nil
}
