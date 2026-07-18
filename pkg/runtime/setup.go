package runtime

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/pkg/executors"
	"github.com/go-openapi/strfmt"
	"golang.org/x/net/context"

	"github.com/ActiveState/cli/internal/artifactcrypto"
	"github.com/ActiveState/cli/internal/chanutils/workerpool"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/httputil"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/proxyreader"
	"github.com/ActiveState/cli/internal/python/wheelinstall"
	"github.com/ActiveState/cli/internal/sliceutils"
	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/platform/api/buildlogstream"
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

	// AuthToken is the platform JWT forwarded to the build-log-streamer WS so
	// the server can authorize the stream. Empty for unauthenticated callers.
	AuthToken string

	// OrgKey is the organization AES-256 key used to decrypt private artifacts
	// during install, with OrgKeyID identifying which key it is. Both are empty
	// when the runtime has no private ingredients.
	OrgKey   []byte
	OrgKeyID string

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

	// skipped records encrypted artifacts that were skipped because no org key
	// was available to decrypt them.
	skipMutex sync.Mutex
	skipped   map[strfmt.UUID]struct{}

	// privateWheels records the depot directories of decrypted private wheels
	// whose installation into site-packages is deferred until every artifact is
	// unpacked, so the Python runtime's site-packages layout can be located on
	// disk rather than guessed.
	privateWheelMutex sync.Mutex
	privateWheels     []string
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
		exists, _ := depot.Exists(a.ArtifactID)
		return !exists
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
		skipped:           map[strfmt.UUID]struct{}{},
	}, nil
}

func (s *setup) markSkipped(id strfmt.UUID) {
	s.skipMutex.Lock()
	defer s.skipMutex.Unlock()
	s.skipped[id] = struct{}{}
}

func (s *setup) wasSkipped(id strfmt.UUID) bool {
	s.skipMutex.Lock()
	defer s.skipMutex.Unlock()
	_, ok := s.skipped[id]
	return ok
}

// skippedAny reports whether any artifact was skipped this run.
func (s *setup) skippedAny() bool {
	s.skipMutex.Lock()
	defer s.skipMutex.Unlock()
	return len(s.skipped) > 0
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

	blog := buildlog.New(s.buildplan.LegacyRecipeID(), s.toBuild, s.opts.AuthToken).
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
			if buildlogstream.IsStreamDenied(err) {
				if s.opts.AuthToken == "" {
					return locale.WrapExternalError(err, "err_buildlog_stream_denied_unauthenticated",
						"Could not monitor in-progress build. Please authenticate by running '[ACTIONABLE]state auth[/RESET]' and try again.")
				}
				return locale.WrapExternalError(err, "err_buildlog_stream_denied_unauthorized",
					"Could not monitor in-progress build. If this is a private project, make sure your account has access to it.")
			}
			return errs.Wrap(err, "errors occurred during buildlog streaming")
		}
	}

	// Wait for workerpool handling build results to finish
	if err := wp.Wait(); err != nil {
		return errs.Wrap(err, "errors occurred during obtain")
	}

	// Install any decrypted private wheels now that every artifact is unpacked,
	// so the Python runtime's site-packages directory can be located on disk.
	if err := s.installPrivateWheels(); err != nil {
		return errs.Wrap(err, "Could not install private wheels")
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

	// Verify checksum.
	if err := s.verifyArtifact(artifact, b); err != nil {
		return errs.Wrap(err, "Artifact checksum validation failed")
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

// verifyArtifact verifies the checksum of the downloaded artifact matches the checksum given by the
// platform, and returns an error if the verification fails.
func (s *setup) verifyArtifact(artifact *buildplan.Artifact, b []byte) error {
	if artifact.Checksum != "" {
		logging.Debug("Validating checksum for %s", artifact.NameAndVersion())
	} else {
		logging.Debug("Skipping checksum validation for %s because the Platform did not provide a checksum to validate against.", artifact.NameAndVersion())
		return nil
	}

	hasher := sha256.New()
	hasher.Write(b)
	checksum := hex.EncodeToString(hasher.Sum(nil))
	artifactChecksum := strings.TrimPrefix(artifact.Checksum, "sha256:")
	if checksum != artifactChecksum {
		logging.Debug("Checksum validation failed. Expected '%s', but was '%s'", artifactChecksum, checksum)
		// Note: the artifact name will be reported higher up the chain
		return locale.NewError("artifact_checksum_failed", "Checksum validation failed")
	}

	return nil
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
	unpackPath := s.depot.Path(artifact.ArtifactID)
	if err := ua.Unarchive(proxy, unpackPath); err != nil {
		if err2 := os.RemoveAll(unpackPath); err2 != nil {
			return errs.Pack(err, errs.Wrap(err2, "unable to remove partially-unpacked directory"))
		}
		return errs.Wrap(err, "unpack failed")
	}

	// Decrypt and extract an encrypted private-ingredient payload, if present.
	outcome, err := s.decryptPayload(artifact.Name(), unpackPath)
	if err != nil {
		if err2 := os.RemoveAll(unpackPath); err2 != nil {
			return errs.Pack(err, errs.Wrap(err2, "unable to remove partially-unpacked directory"))
		}
		return errs.Wrap(err, "decrypt failed")
	}
	if outcome == decryptSkipped {
		s.markSkipped(artifact.ArtifactID)
		if err := os.RemoveAll(unpackPath); err != nil {
			return errs.Wrap(err, "unable to remove skipped artifact directory")
		}
		logging.Warning("Skipping encrypted artifact %s (%s): no org key available", artifact.ArtifactID, artifact.Name())
		return nil
	}

	if err := s.depot.Put(artifact.ArtifactID); err != nil {
		return errs.Wrap(err, "Could not put artifact in depot")
	}
	if outcome == decryptDone {
		logging.Debug("Decrypted private artifact %s (%s)", artifact.ArtifactID, artifact.Name())
		if err := s.depot.MarkPrivate(artifact.ArtifactID); err != nil {
			return errs.Wrap(err, "Could not mark decrypted artifact as private")
		}
		switch {
		case s.isPrivateWheel(unpackPath):
			// Defer installation into site-packages until all artifacts are
			// unpacked; only then can we locate the Python runtime's
			// site-packages layout on disk (see installPrivateWheels).
			s.recordPrivateWheel(unpackPath)
		default:
			multilog.Error("Decrypted private artifact %s (%s) is of an unknown type; cannot install it", artifact.ArtifactID, artifact.Name())
		}
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

type decryptOutcome int

const (
	decryptNotEncrypted decryptOutcome = iota // no encrypted payload present
	decryptDone                               // payload decrypted and extracted in place
	decryptSkipped                            // encrypted, but no org key available
)

// decryptPayload finds the encrypted private-ingredient payload within the
// unpacked artifact (located by name and confirmed by envelope magic),
// decrypts it, and extracts the inner tar.gz archive in place of the
// ciphertext so it lands where the runtime.json points.
//
// A missing key returns decryptSkipped; a wrong key or corrupt payload returns
// an error.
func (s *setup) decryptPayload(artifactName, unpackPath string) (outcome decryptOutcome, rerr error) {
	payloadPath, err := findEncryptedPayload(unpackPath)
	if err != nil {
		return decryptNotEncrypted, errs.Wrap(err, "could not scan for encrypted payload")
	}
	if payloadPath == "" {
		return decryptNotEncrypted, nil
	}
	logging.Debug("Detected encrypted payload in artifact %s", artifactName)

	if len(s.opts.OrgKey) == 0 {
		return decryptSkipped, nil
	}

	// Confirm the key matches the payload header.
	header, err := readPayloadHeader(payloadPath)
	if err != nil {
		return decryptNotEncrypted, errs.Wrap(err, "could not read encrypted payload header")
	}
	if err := header.CheckKey(s.opts.OrgKey); err != nil {
		return decryptNotEncrypted, errs.Wrap(err, "org key does not match encrypted artifact %s", artifactName)
	}

	// Decrypt to a private temp dir, then extract the archive in place.
	tmpDir, err := os.MkdirTemp(filepath.Dir(unpackPath), ".decrypt-")
	if err != nil {
		return decryptNotEncrypted, errs.Wrap(err, "could not create decrypt temp dir")
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			rerr = errs.Pack(rerr, errs.Wrap(err, "could not remove decrypt temp dir"))
		}
	}()

	archivePath := filepath.Join(tmpDir, "payload")
	src, err := os.Open(payloadPath)
	if err != nil {
		return decryptNotEncrypted, errs.Wrap(err, "could not open encrypted payload")
	}
	err = artifactcrypto.Decrypt(src, archivePath, s.opts.OrgKey)
	if cerr := src.Close(); cerr != nil {
		err = errs.Pack(err, errs.Wrap(cerr, "could not close encrypted payload"))
	}
	if err != nil {
		return decryptNotEncrypted, errs.Wrap(err, "could not decrypt artifact %s", artifactName)
	}

	// Remove the ciphertext from the artifact directory.
	if err := os.Remove(payloadPath); err != nil {
		return decryptNotEncrypted, errs.Wrap(err, "could not remove ciphertext")
	}

	archive, err := os.Open(archivePath)
	if err != nil {
		return decryptNotEncrypted, errs.Wrap(err, "could not open decrypted payload")
	}
	defer func() {
		if err := archive.Close(); err != nil {
			rerr = errs.Pack(rerr, errs.Wrap(err, "could not close decrypted payload"))
		}
	}()
	// Extract alongside the ciphertext so the decrypted contents land where the
	// runtime.json points.
	archiveUA := unarchiver.NewTarGz(unarchiver.WithUntrustedSource())
	if err := archiveUA.Unarchive(archive, filepath.Dir(payloadPath)); err != nil {
		return decryptNotEncrypted, errs.Wrap(err, "could not extract decrypted artifact %s", artifactName)
	}

	// Restrict the decrypted artifact directory to owner-only (0700).
	if err := os.Chmod(unpackPath, 0700); err != nil {
		return decryptNotEncrypted, errs.Wrap(err, "could not restrict decrypted artifact directory")
	}

	return decryptDone, nil
}

// findEncryptedPayload returns the path of the encrypted private payload within
// dir, searched recursively, or "" if none is present. The payload is located by
// its conventional name (artifactcrypto.PayloadFilename) and confirmed by its
// artifactcrypto envelope magic, so a plaintext file that happens to share the
// name is ignored.
func findEncryptedPayload(dir string) (string, error) {
	var found string
	walkErr := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || d.Name() != artifactcrypto.PayloadFilename {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return errs.Wrap(err, "could not open artifact file")
		}
		encrypted, err := artifactcrypto.IsEncrypted(f)
		if cerr := f.Close(); cerr != nil {
			err = errs.Pack(err, errs.Wrap(cerr, "could not close artifact file"))
		}
		if err != nil {
			return errs.Wrap(err, "could not detect encrypted payload")
		}
		if encrypted {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if walkErr != nil {
		return "", errs.Wrap(walkErr, "could not scan artifact directory")
	}
	return found, nil
}

func readPayloadHeader(path string) (header artifactcrypto.Header, rerr error) {
	f, err := os.Open(path)
	if err != nil {
		return artifactcrypto.Header{}, errs.Wrap(err, "could not open encrypted payload")
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			rerr = errs.Pack(rerr, errs.Wrap(cerr, "could not close encrypted payload"))
		}
	}()
	return artifactcrypto.ParseHeader(f)
}

// isPrivateWheel reports whether the decrypted payload under dir is a wheel. We
// control the payload via `state publish --build`, so a .whl extension is a
// sufficient test.
func (s *setup) isPrivateWheel(dir string) bool {
	wheelPath, err := findWheel(dir)
	return err == nil && wheelPath != ""
}

// recordPrivateWheel notes a decrypted private wheel's depot directory for
// deferred installation into site-packages once all artifacts are unpacked.
func (s *setup) recordPrivateWheel(dir string) {
	s.privateWheelMutex.Lock()
	defer s.privateWheelMutex.Unlock()
	s.privateWheels = append(s.privateWheels, dir)
}

// installPrivateWheels installs each recorded private wheel into the Python
// runtime's site-packages directory. It runs after every artifact is unpacked
// so the site-packages location -- which varies by platform and Python version
// (e.g. lib/pythonX.Y/site-packages on Linux, lib/site-packages on macOS) -- can
// be discovered on disk rather than guessed.
func (s *setup) installPrivateWheels() error {
	if len(s.privateWheels) == 0 {
		return nil
	}

	relSitePackages, err := s.locateSitePackages()
	if err != nil {
		return errs.Wrap(err, "could not locate the Python runtime's site-packages directory")
	}

	for _, dir := range s.privateWheels {
		if err := s.installPrivateWheel(dir, relSitePackages); err != nil {
			return errs.Wrap(err, "could not install private wheel in %s", dir)
		}
	}
	return nil
}

// installPrivateWheel installs the decrypted wheel found under artifactDir into
// the runtime's site-packages directory (relSitePackages, relative to the
// install root) and adds that directory to PYTHONPATH in the artifact's
// runtime.json. Laying the wheel out at the same relative path the Python
// runtime uses means the deploy merges its contents into
// ${INSTALLDIR}/<relSitePackages>, where Python already resolves them.
func (s *setup) installPrivateWheel(artifactDir, relSitePackages string) error {
	wheelPath, err := findWheel(artifactDir)
	if err != nil {
		return errs.Wrap(err, "could not locate decrypted wheel")
	}
	if wheelPath == "" {
		return errs.New("decrypted private artifact contains no wheel")
	}

	rtPath := filepath.Join(artifactDir, envdef.EnvironmentDefinitionFilename)
	envDef, err := envdef.NewEnvironmentDefinition(rtPath)
	if err != nil {
		return errs.Wrap(err, "could not load runtime definition")
	}

	// InstallDir is the subtree of the artifact that gets deployed onto the
	// install root, so the site-packages must sit beneath it for the deploy to
	// place it at ${INSTALLDIR}/<relSitePackages>.
	sitePackages := filepath.Join(artifactDir, envDef.InstallDir, relSitePackages)
	if err := wheelinstall.Install(wheelPath, sitePackages); err != nil {
		return errs.Wrap(err, "could not install wheel")
	}
	if err := os.Remove(wheelPath); err != nil {
		return errs.Wrap(err, "could not remove installed wheel")
	}

	// Match the Python runtime's own PYTHONPATH directives (":" separator,
	// inherit=false) or the cross-artifact environment merge fails with
	// "incompatible separator or inherit directives".
	envDef.Env = append(envDef.Env, envdef.EnvironmentVariable{
		Name:      "PYTHONPATH",
		Values:    []string{"${INSTALLDIR}/" + filepath.ToSlash(relSitePackages)},
		Join:      envdef.Prepend,
		Inherit:   false,
		Separator: ":", // OS-independent
	})
	if err := envDef.Save(artifactDir); err != nil {
		return errs.Wrap(err, "could not save runtime definition")
	}
	return nil
}

// locateSitePackages discovers the Python runtime's site-packages directory by
// scanning the unpacked non-private artifacts, and returns its path relative to
// the install root. Searching for the directory that the Python runtime already
// provides keeps us correct across platforms and Python versions instead of
// hard-coding a layout.
func (s *setup) locateSitePackages() (string, error) {
	// Collect the install roots of every candidate artifact (all non-private
	// artifacts that deploy something). The Python runtime is among them.
	var installRoots []string
	for _, artifact := range s.buildplan.Artifacts() {
		artifactDir := s.depot.Path(artifact.ArtifactID)
		if !fileutils.DirExists(artifactDir) {
			continue
		}
		// Skip decrypted private artifacts: they are the wheels being installed,
		// not the Python runtime that owns site-packages.
		if _, info := s.depot.Exists(artifact.ArtifactID); info != nil && info.Private {
			continue
		}
		envDef, err := envdef.NewEnvironmentDefinition(filepath.Join(artifactDir, envdef.EnvironmentDefinitionFilename))
		if err != nil {
			continue // no runtime.json; nothing this artifact deploys can be site-packages
		}
		installRoots = append(installRoots, filepath.Join(artifactDir, envDef.InstallDir))
	}

	// First try the known layouts via cheap globs. Only if none of them match
	// (an unexpected layout) do we fall back to a full walk of every root.
	candidates := globSitePackages(installRoots)
	if len(candidates) == 0 {
		var err error
		candidates, err = walkSitePackages(installRoots)
		if err != nil {
			return "", errs.Wrap(err, "could not scan for site-packages")
		}
	}

	if len(candidates) == 0 {
		return "", errs.New("no Python site-packages directory found in the runtime; a Python runtime is required to install a private wheel")
	}

	// Prefer the shortest path (the canonical purelib site-packages), then order
	// lexicographically, so the choice is deterministic.
	sort.Slice(candidates, func(i, j int) bool {
		if len(candidates[i]) != len(candidates[j]) {
			return len(candidates[i]) < len(candidates[j])
		}
		return candidates[i] < candidates[j]
	})
	if len(candidates) > 1 {
		logging.Debug("Multiple site-packages directories found (%v); using %s", candidates, candidates[0])
	}
	return candidates[0], nil
}

// sitePackagesGlobs are the known relative locations of a Python runtime's
// site-packages directory, with a wildcard for the version-specific segment.
// They cover the Linux, macOS, and Windows layouts the platform produces.
var sitePackagesGlobs = []string{
	filepath.Join("lib", "python*", "site-packages"),
	filepath.Join("lib64", "python*", "site-packages"),
	filepath.Join("lib", "site-packages"),
	filepath.Join("usr", "lib", "python*", "site-packages"),
	filepath.Join("usr", "lib64", "python*", "site-packages"),
	filepath.Join("Lib", "site-packages"),
}

// globSitePackages returns the distinct site-packages directories (relative to
// their root) found under the given roots using the known glob patterns.
func globSitePackages(roots []string) []string {
	var candidates []string
	seen := map[string]struct{}{}
	for _, root := range roots {
		for _, glob := range sitePackagesGlobs {
			matches, err := filepath.Glob(filepath.Join(root, glob))
			if err != nil {
				continue // only ErrBadPattern, and our patterns are static
			}
			for _, match := range matches {
				if !fileutils.DirExists(match) {
					continue
				}
				rel, err := filepath.Rel(root, match)
				if err != nil {
					continue
				}
				if _, ok := seen[rel]; !ok {
					seen[rel] = struct{}{}
					candidates = append(candidates, rel)
				}
			}
		}
	}
	return candidates
}

// walkSitePackages returns the distinct site-packages directories (relative to
// their root) found by fully walking each root. It is the fallback for layouts
// the globs do not cover.
func walkSitePackages(roots []string) ([]string, error) {
	var candidates []string
	seen := map[string]struct{}{}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && d.Name() == "site-packages" {
				rel, err := filepath.Rel(root, path)
				if err != nil {
					return err
				}
				if _, ok := seen[rel]; !ok {
					seen[rel] = struct{}{}
					candidates = append(candidates, rel)
				}
				return filepath.SkipDir
			}
			return nil
		})
		if err != nil {
			return nil, errs.Wrap(err, "could not scan %s for site-packages", root)
		}
	}
	return candidates, nil
}

// findWheel returns the path of the single .whl under dir (searched recursively),
// or "" if none is present.
func findWheel(dir string) (string, error) {
	var found string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".whl") {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", errs.Wrap(err, "could not scan for wheel")
	}
	return found, nil
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

	// Artifacts skipped during unpack are not in the depot. Report the skip so
	// the install progress bar still accounts for them.
	if s.wasSkipped(id) {
		if err := s.fireEvent(events.ArtifactInstallSkipped{id, artifact.Name()}); err != nil {
			return errs.Wrap(err, "Could not handle ArtifactInstallSkipped event")
		}
		return nil
	}

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

		err = s.depot.Track(artifact, &deployment{
			Type:  deploymentTypeEcosystem,
			Path:  s.path,
			Files: files,
		})
		if err != nil {
			return errs.Wrap(err, "Could not track deployment")
		}
		return nil
	}

	envDef, err := s.env.Load(artifactDepotPath)
	if err != nil {
		return errs.Wrap(err, "Could not get env")
	}

	var deploy *deployment
	if envDef.NeedsTransforms() || !s.supportsHardLinks || s.opts.Portable {
		deploy, err = s.depot.DeployViaCopy(id, envDef.InstallDir, s.path)
		if err != nil {
			return errs.Wrap(err, "Could not deploy artifact via copy")
		}
		if envDef.NeedsTransforms() {
			if err := envDef.ApplyFileTransforms(s.path); err != nil {
				return errs.Wrap(err, "Could not apply env transforms")
			}
		}
	} else {
		deploy, err = s.depot.DeployViaLink(id, envDef.InstallDir, s.path)
		if err != nil {
			return errs.Wrap(err, "Could not deploy artifact via link")
		}
	}
	err = s.depot.Track(artifact, deploy)
	if err != nil {
		return errs.Wrap(err, "Could not track deployment")
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

	envDef, err := s.env.Load(artifactDepotPath)
	if err != nil {
		return errs.Wrap(err, "Could not get env")
	}

	if err := s.env.Unload(artifactDepotPath); err != nil {
		return errs.Wrap(err, "Could not unload artifact envdef")
	}

	// If this is a dynamically imported artifact, tell the ecosystem to remove/undeploy it.
	if exists, artifact := s.depot.Exists(id); exists && artifact != nil && artifact.Namespace != "" {
		if ecosys := filterEcosystemMatchingNamespace(s.ecosystems, artifact.Namespace); ecosys != nil {
			installedFiles := []string{}
			// Find record of our deployment
			deployments := sliceutils.Filter(s.depot.Deployments(id), func(d deployment) bool { return d.Path == s.path })
			if len(deployments) > 0 {
				installedFiles = deployments[0].Files
			}

			// Convert relative install locations to absolute paths.
			for i, file := range installedFiles {
				installedFiles[i] = filepath.Join(s.path, file)
			}

			// Remove/undeploy the artifact.
			err := ecosys.Remove(artifact.Name, artifact.Version, installedFiles)
			if err != nil {
				return errs.Wrap(err, "Ecosystem unable to remove artifact")
			}
			s.depot.Untrack(id, filepath.Join(s.path, id.String()))
			return nil
		}
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
