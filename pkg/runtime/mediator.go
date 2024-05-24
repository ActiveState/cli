package runtime

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/runtime/internal/buildlog"
	"github.com/go-openapi/strfmt"
	"golang.org/x/net/context"
)

// setupPlan is responsible for funneling the setup through the multiple sources that we can obtain artifacts from
// For example in order to obtain a build status we may rely on the initial buildplan, or we may rely on the buildlog
// streamer if the artifact is still being built.
type setupPlan struct {
	// toBuild encompasses all artifacts that will need to be toBuild for this runtime.
	// This does NOT mean every artifact in the runtime closure if this is an update (as oppose to a fresh toInstall).
	// Because when we update we likely already have some of the requisite artifacts installed, and thus we don't need their toBuild.
	toBuild buildplan.ArtifactIDMap

	// toDownload encompasses all artifacts that will need to be downloaded for this runtime. The same caveat applies as for toBuild.
	toDownload buildplan.ArtifactIDMap

	// toDownloadNow is a subset of toDownload. It covers any artifacts that do not need to be first toBuild.
	// Everything in toDownloadNow will also appear in toDownload.
	toDownloadNow buildplan.ArtifactIDMap // toDownloadNow are artifacts that can be downloaded straight away, no need to wait for a toBuild

	// toInstall encompasses all artifacts that will need to be installed for this runtime. The same caveat applies as for toBuild.
	toInstall buildplan.ArtifactIDMap

	// toInstallNow is a subset of toInstall. It covers any artifacts that can to be installed straight away, no need to wait for a toBuild or a toDownload.
	toInstallNow buildplan.ArtifactIDMap
}

func (r *Runtime) calculateSetupPlan(bp *buildplan.BuildPlan, platformID strfmt.UUID) (*setupPlan, error) {
	// Start off with the full range of artifacts relevant to our platform
	installableArtifacts := bp.Artifacts(
		buildplan.FilterPlatformArtifacts(platformID),
		buildplan.FilterRuntimeArtifacts(),
		buildplan.FilterStateArtifacts(),
	)

	// Identify which artifacts we'll need to install, this filters out any artifacts that are already installed.
	artifactsToInstall := installableArtifacts.Filter(func(a *buildplan.Artifact) bool {
		_, installed := r.storedArtifacts[a.ArtifactID]
		return !installed
	})

	// Calculate which artifacts need to be downloaded; if an artifact we want to install is not in our depot then
	// by definition we'll need to download it.
	// We also calculate which artifacts are immediately ready to be installed, as its the inverse condition of the above.
	artifactsToDownload := buildplan.Artifacts{}
	artifactsToInstallNow := buildplan.Artifacts{}
	for _, a := range artifactsToInstall {
		if _, cached := r.depotArtifacts[a.ArtifactID]; cached {
			artifactsToInstallNow = append(artifactsToInstallNow, a)
		} else {
			artifactsToDownload = append(artifactsToDownload, a)
		}
	}

	// Now that we know which artifacts we'll need to download we can use this as our basis for calculating which artifacts
	// still need to be build. This encompasses the artifacts themselves, as well as any of their dependencies. And of
	// course we only want to filter artifacts that actually require a build, as the build may be cached server side.
	artifactsToBuild := append(artifactsToDownload, artifactsToDownload.Dependencies(true)...).Filter(buildplan.FilterNeedsBuild())
	artifactsToBuild = sliceutils.UniqueByProperty(artifactsToBuild, func(a *buildplan.Artifact) any { return a.ArtifactID })

	return &setupPlan{
		toBuild:       artifactsToBuild.ToIDMap(),
		toDownload:    artifactsToDownload.ToIDMap(),
		toDownloadNow: artifactsToDownload.Filter(buildplan.FilterSuccessfulArtifacts()).ToIDMap(),
		toInstall:     artifactsToInstall.ToIDMap(),
		toInstallNow:  artifactsToInstallNow.ToIDMap(),
	}, nil
}

func (m *setupPlan) mediateAndWait() error {
	blog, err := buildlog.New(context.Background(), m.toBuild, r.opts.EventHandlers, bp.RecipeID(), filepath.Join(r.path, configDir, buildLogFile))
}
