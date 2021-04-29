package alternative

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
	"github.com/thoas/go-funk"
)

type Setup struct {
	artifacts artifact.ArtifactRecipeMap
	store     *store.Store
}

func NewSetup(store *store.Store, artifacts artifact.ArtifactRecipeMap) *Setup {
	return &Setup{store: store, artifacts: artifacts}
}

func (s *Setup) ReusableArtifacts(changeset artifact.ArtifactChangeset, storedArtifacted store.StoredArtifactMap) store.StoredArtifactMap {
	keep := make(store.StoredArtifactMap)
	// copy store
	for k, v := range storedArtifacted {
		keep[k] = v
	}

	// remove all updated and removed artifacts
	for _, upd := range changeset.Updated {
		delete(keep, upd.FromID)
	}
	for _, id := range changeset.Removed {
		delete(keep, id)
	}

	return keep
}

func (s *Setup) DeleteOutdatedArtifacts(changeset artifact.ArtifactChangeset, storedArtifacted, alreadyInstalled store.StoredArtifactMap) error {
	del := map[artifact.ArtifactID]struct{}{}
	for _, upd := range changeset.Updated {
		del[upd.FromID] = struct{}{}
	}
	for _, id := range changeset.Removed {
		del[id] = struct{}{}
	}

	// sort files and dirs in keep for faster look-up
	for _, artf := range alreadyInstalled {
		sort.Strings(artf.Dirs)
		sort.Strings(artf.Files)
	}

	for _, artf := range storedArtifacted {
		if _, deleteMe := del[artf.ArtifactID]; !deleteMe {
			continue
		}

		for _, file := range artf.Files {
			if !fileutils.TargetExists(file) {
				continue // don't care it's already deleted (might have been deleted by another artifact that supplied the same file)
			}
			if artifactsContainFile(file, alreadyInstalled) {
				continue
			}
			if err := os.Remove(file); err != nil {
				return locale.WrapError(err, "err_rm_artf", "Could not remove old package file at {{.V0}}.", file)
			}
		}

		dirs := artf.Dirs
		sort.Slice(dirs, func(i, j int) bool {
			return dirs[i] > dirs[j]
		})

		for _, dir := range dirs {
			if !fileutils.DirExists(dir) {
				continue
			}

			deleteOk, err := dirCanBeDeleted(dir, alreadyInstalled)
			if err != nil {
				logging.Error("Could not determine if directory %s could be deleted: %v", dir, err)
				continue
			}
			if !deleteOk {
				continue
			}

			err = os.RemoveAll(dir)
			if err != nil {
				return locale.WrapError(err, "err_rm_artf_dir", "Could not remove empty artifact directory at {{.V0}}", dir)
			}
		}

		if err := s.store.DeleteArtifactStore(artf.ArtifactID); err != nil {
			return errs.Wrap(err, "Could not delete artifact store")
		}
	}

	return nil
}

// dirCanBeDeleted checks if the given directory is empty - ignoring files and sub-directories that
// are not in the cache.
func dirCanBeDeleted(dir string, cache map[artifact.ArtifactID]store.StoredArtifact) (bool, error) {
	if artifactsContainDir(dir, cache) {
		return false, nil
	}

	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return false, errs.Wrap(err, "Could not read directory.")
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if artifactsContainDir(filepath.Join(dir, entry.Name()), cache) {
				return false, nil
			}
		} else {
			if artifactsContainFile(filepath.Join(dir, entry.Name()), cache) {
				return false, nil
			}
		}
	}
	return true, nil
}

func sortedStringSliceContains(slice []string, x string) bool {
	i := sort.SearchStrings(slice, x)
	return i != len(slice) && slice[i] == x
}

func artifactsContainDir(dir string, artifactCache map[artifact.ArtifactID]store.StoredArtifact) bool {
	for _, v := range artifactCache {
		if funk.Contains(v.Dirs, dir) {
			return true
		}
	}
	return false
}

func artifactsContainFile(file string, artifactCache map[artifact.ArtifactID]store.StoredArtifact) bool {
	for _, v := range artifactCache {
		if sortedStringSliceContains(v.Files, file) {
			return true
		}
	}
	return false
}

func (s *Setup) ResolveArtifactName(a artifact.ArtifactID) string {
	if artf, ok := s.artifacts[a]; ok {
		return artf.Name
	}
	return locale.Tl("alternative_unknown_pkg_name", "unknown")
}

func (s *Setup) DownloadsFromBuild(buildStatus *headchef_models.BuildStatusResponse) ([]artifact.ArtifactDownload, error) {
	return artifact.NewDownloadsFromBuild(buildStatus)
}
