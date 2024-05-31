package runtime

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"

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
	"github.com/ActiveState/cli/pkg/runtime/executors"
	"github.com/ActiveState/cli/pkg/runtime/internal/buildlog"
	"github.com/ActiveState/cli/pkg/runtime/internal/envdef"
	"github.com/ActiveState/cli/pkg/sysinfo"
	"github.com/go-openapi/strfmt"
	"golang.org/x/net/context"
)

type Opts struct {
	PreferredLibcVersion string
	EventHandlers        []events.HandlerFunc
	BuildlogFilePath     string

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
	path      string
	opts      *Opts
	depot     *depot
	env       *envdef.Collection
	buildplan *buildplan.BuildPlan

	// toBuild encompasses all artifacts that will need to be build for this runtime.
	// This does NOT mean every artifact in the runtime closure if this is an update (as oppose to a fresh toInstall).
	// Because when we update we likely already have some of the requisite artifacts installed, and thus we don't need their toBuild.
	toBuild buildplan.ArtifactIDMap

	// toDownload encompasses all artifacts that will need to be downloaded for this runtime. The same caveat applies as for toBuild.
	toDownload buildplan.ArtifactIDMap

	// toInstall encompasses all artifacts that will need to be installed for this runtime. The same caveat applies as for toBuild.
	toInstall buildplan.ArtifactIDMap

	// toUninstall encompasses all artifacts that will need to be uninstalled for this runtime.
	toUninstall map[strfmt.UUID]struct{}
}

func newSetup(path string, bp *buildplan.BuildPlan, opts *Opts) (*setup, error) {
	env := envdef.NewCollection()

	depot, err := newDepot(env)
	if err != nil {
		return nil, errs.Wrap(err, "Could not create depot")
	}

	installedArtifacts := depot.List(path)

	platformID, err := model.FilterCurrentPlatform(sysinfo.OS().String(), bp.Platforms(), opts.PreferredLibcVersion)
	if err != nil {
		return nil, ErrNoPlatformMatch
	}

	// Start off with the full range of artifacts relevant to our platform
	installableArtifacts := bp.Artifacts(
		buildplan.FilterPlatformArtifacts(platformID),
		buildplan.FilterRuntimeArtifacts(),
		buildplan.FilterStateArtifacts(),
	)

	// Identify which artifacts we'll need to install, this filters out any artifacts that are already installed.
	artifactsToInstall := installableArtifacts.Filter(func(a *buildplan.Artifact) bool {
		_, installed := installedArtifacts[a.ArtifactID]
		return !installed
	})

	// Identify which artifacts we can uninstall
	installableArtifactsMap := installableArtifacts.ToIDMap()
	artifactsToUninstall := map[strfmt.UUID]struct{}{}
	for id := range installedArtifacts {
		if _, required := installableArtifactsMap[id]; !required {
			artifactsToUninstall[id] = struct{}{}
		}
	}

	// Calculate which artifacts need to be downloaded; if an artifact we want to install is not in our depot then
	// by definition we'll need to download it.
	// We also calculate which artifacts are immediately ready to be installed, as its the inverse condition of the above.
	artifactsToDownload := artifactsToInstall.Filter(func(a *buildplan.Artifact) bool {
		return !depot.Exists(a.ArtifactID)
	})

	// Now that we know which artifacts we'll need to download we can use this as our basis for calculating which artifacts
	// still need to be build. This encompasses the artifacts themselves, as well as any of their dependencies. And of
	// course we only want to filter artifacts that actually require a build, as the build may be cached server side.
	artifactsToBuild := append(artifactsToDownload, artifactsToDownload.Dependencies(true)...).Filter(buildplan.FilterNeedsBuild())
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

	return &setup{
		path:        path,
		opts:        opts,
		env:         env,
		depot:       depot,
		buildplan:   bp,
		toBuild:     artifactsToBuild.ToIDMap(),
		toDownload:  artifactsToDownload.ToIDMap(),
		toInstall:   artifactsToInstall.ToIDMap(),
		toUninstall: artifactsToUninstall,
	}, nil
}

func (s *setup) RunAndWait() (rerr error) {
	defer func() {
		// Handle success / failure event
		var name = "success"
		var ev events.Event = events.Success{}
		if rerr != nil {
			name = "failure"
			ev = events.Failure{}
		}

		err := s.fireEvent(ev)
		if err != nil {
			rerr = errs.Pack(rerr, errs.Wrap(err, "Could not handle %s event", name))
		}
	}()

	if err := s.fireEvent(events.Start{
		RecipeID:            s.buildplan.RecipeID(),
		RequiresBuild:       s.buildplan.IsBuildInProgress() && len(s.toDownload) > 0,
		LogFilePath:         s.opts.BuildlogFilePath,
		ArtifactsToBuild:    s.toBuild,
		ArtifactsToDownload: s.toDownload,
		ArtifactsToInstall:  s.toInstall,
	}); err != nil {
		return errs.Wrap(err, "Could not handle Start event")
	}

	if err := s.update(); err != nil {
		return errs.Wrap(err, "Could not update")
	}

	// Ensure our collection has all our artifacts
	// Technically this is redundant as the depot would've already hit these, but it's better not to rely
	// on implicit behavior of other packages to achieve the results we want in this one, and it's cached anyway so
	// the performance impact is trivial.
	for id := range s.depot.List(s.path) {
		_, err := s.env.Get(s.depot.Path(id))
		if err != nil {
			return errs.Wrap(err, "Could not get env")
		}
	}

	if err := s.save(); err != nil {
		return errs.Wrap(err, "Could not save runtime config")
	}

	return nil
}

func (s *setup) update() error {
	blog := buildlog.New(s.buildplan.RecipeID(), s.toBuild).
		WithEventHandler(s.opts.EventHandlers...).
		WithLogFile(filepath.Join(s.path, configDir, buildLogFile))

	// Download artifacts when ready
	wp := workerpool.New(maxConcurrency)
	for _, a := range s.toDownload {
		s.onArtifactBuildReady(blog, a, func(artifact *buildplan.Artifact) {
			wp.Submit(func() error {
				if err := s.obtain(artifact); err != nil {
					return errs.Wrap(err, "download failed")
				}
				return nil
			})
		})
	}

	// Wait for build to finish
	if !s.buildplan.IsBuildReady() {
		if err := blog.Wait(context.Background()); err != nil {
			return errs.Wrap(err, "errors occurred during buildlog streaming")
		}
	}

	// Wait for workerpool handling build results to finish
	if err := wp.Wait(); err != nil {
		return errs.Wrap(err, "errors occurred during obtain")
	}

	// Now we start modifying the runtime directory
	// This happens AFTER all the download steps are finished, and should be extremely fast because installing is
	// simply creating links to the depot.
	// We do this as a separate step so we don't leave the runtime dir in a half-installed state if issues happen earlier
	// on in the process.

	// Uninstall artifacts
	for id := range s.toUninstall {
		if err := s.depot.Undeploy(id, s.path); err != nil {
			return errs.Wrap(err, "Could not unlink artifact")
		}
	}

	// Install artifacts
	wp = workerpool.New(maxConcurrency)
	for _, a := range s.toInstall {
		func(a *buildplan.Artifact) { // We can get rid of this once we upgrade to Go 1.22 -- https://go.dev/blog/loopvar-preview
			wp.Submit(func() error {
				if err := s.install(a.ArtifactID); err != nil {
					return errs.Wrap(err, "Could not install artifact")
				}
				return nil
			})
		}(a)
	}

	// Wait for workerpool handling artifact installs to finish
	if err := wp.Wait(); err != nil {
		return errs.Wrap(err, "errors occurred during install")
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

func (s *setup) onArtifactBuildReady(blog *buildlog.BuildLog, artifact *buildplan.Artifact, cb func(*buildplan.Artifact)) {
	if _, ok := s.toBuild[artifact.ArtifactID]; !ok {
		// No need to build, artifact can already be downloaded
		cb(artifact)
		return
	}

	blog.OnArtifactReady(artifact.ArtifactID, cb)
}

func (s *setup) obtain(artifact *buildplan.Artifact) (rerr error) {
	// Download artifact
	b, err := s.download(artifact)
	if err != nil {
		return errs.Wrap(err, "download failed")
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
		}
	}()

	var ua unarchiver.Unarchiver = unarchiver.NewTarGz()
	if strings.HasSuffix(strings.ToLower(artifact.URL), "zip") {
		ua = unarchiver.NewZip()
	}

	if err := s.fireEvent(events.ArtifactUnpackStarted{artifact.ArtifactID, len(b)}); err != nil {
		return errs.Wrap(err, "Could not handle ArtifactUnpackStarted event")
	}

	var numUnpackedFiles int
	ua.SetNotifier(func(_ string, _ int64, isDir bool) {
		if !isDir {
			numUnpackedFiles++
		}
	})

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

	if err := s.fireEvent(events.ArtifactUnpackSuccess{artifact.ArtifactID}); err != nil {
		return errs.Wrap(errs.Pack(err, err), "Could not handle ArtifactUnpackSuccess event")
	}

	return nil
}

func (s *setup) updateExecutors() error {
	execPath := filepath.Join(s.path, configDir, executorDir)
	if err := fileutils.MkdirUnlessExists(execPath); err != nil {
		return errs.Wrap(err, "Could not create executors directory")
	}

	env, err := s.env.Environment()
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

func (s *setup) save() error {
	env, err := s.env.Environment()
	if err != nil {
		return errs.Wrap(err, "Could not get env")
	}
	envB, err := json.Marshal(env)
	if err != nil {
		return errs.Wrap(err, "Could not marshal env")
	}
	if err := fileutils.WriteFile(filepath.Join(s.path, configDir, environmentFile), envB); err != nil {
		return errs.Wrap(err, "Could not write environment file")
	}

	return nil
}

func (s *setup) install(id strfmt.UUID) (rerr error) {
	defer func() {
		if rerr == nil {
			if err := s.fireEvent(events.ArtifactInstallSuccess{}); err != nil {
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
	if err := s.depot.Deploy(id, s.path); err != nil {
		return errs.Wrap(err, "Could not link artifact")
	}
	return nil
}
