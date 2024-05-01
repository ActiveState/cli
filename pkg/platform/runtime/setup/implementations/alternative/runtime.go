package alternative

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
	"github.com/go-openapi/strfmt"
	"github.com/thoas/go-funk"
)

type Setup struct {
	store *store.Store
}

func NewSetup(store *store.Store) *Setup {
	return &Setup{store: store}
}

func (s *Setup) DeleteOutdatedArtifacts(changeset *buildplan.ArtifactChangeset, storedArtifacted, alreadyInstalled store.StoredArtifactMap) error {
	if changeset == nil {
		return nil
	}

	del := map[strfmt.UUID]struct{}{}
	for _, upd := range changeset.Updated {
		del[upd.From.ArtifactID] = struct{}{}
	}
	for _, rem := range changeset.Removed {
		del[rem.ArtifactID] = struct{}{}
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
				multilog.Error("Could not determine if directory %s could be deleted: %v", dir, err)
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
func dirCanBeDeleted(dir string, cache map[strfmt.UUID]store.StoredArtifact) (bool, error) {
	if artifactsContainDir(dir, cache) {
		return false, nil
	}

	entries, err := os.ReadDir(dir)
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

func artifactsContainDir(dir string, artifactCache map[strfmt.UUID]store.StoredArtifact) bool {
	for _, v := range artifactCache {
		if funk.Contains(v.Dirs, dir) {
			return true
		}
	}
	return false
}

func artifactsContainFile(file string, artifactCache map[strfmt.UUID]store.StoredArtifact) bool {
	for _, v := range artifactCache {
		if sortedStringSliceContains(v.Files, file) {
			return true
		}
	}
	return false
}

func (s *Setup) ResolveArtifactName(a strfmt.UUID) string {
	return locale.T("alternative_unknown_pkg_name")
}
