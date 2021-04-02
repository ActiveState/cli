package alternative

import (
	"os"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/pkg/platform/api/headchef/headchef_models"
	"github.com/ActiveState/cli/pkg/platform/runtime/artifact"
	"github.com/ActiveState/cli/pkg/platform/runtime/store"
	"github.com/go-openapi/strfmt"
)

type Setup struct {
	artifacts artifact.ArtifactRecipeMap
	store     *store.Store
}

func NewSetup(store *store.Store, artifacts artifact.ArtifactRecipeMap) *Setup {
	return &Setup{store: store, artifacts: artifacts}
}

func (s *Setup) DeleteOutdatedArtifacts(changeset artifact.ArtifactChangeset, storedArtifacted store.StoredArtifactMap) error {
	del := map[strfmt.UUID]struct{}{}
	for _, upd := range changeset.Updated {
		del[upd.FromID] = struct{}{}
	}
	for _, id := range changeset.Removed {
		del[id] = struct{}{}
	}

	for _, artf := range storedArtifacted {
		if _, deleteMe := del[artf.ArtifactID]; !deleteMe {
			continue
		}

		for _, file := range artf.Files {
			if !fileutils.TargetExists(file) {
				continue // don't care it's already deleted (might have been deleted by another artifact that supplied the same file)
			}
			if err := os.Remove(file); err != nil {
				return locale.WrapError(err, "err_rm_artf", "", "Could not remove old package file at {{.V0}}.", file)
			}
		}

		if err := s.store.DeleteArtifactStore(artf.ArtifactID); err != nil {
			return errs.Wrap(err, "Could not delete artifact store")
		}
	}

	return nil
}

func (s *Setup) ResolveArtifactName(a artifact.ArtifactID) string {
	if artf, ok := s.artifacts[a]; ok {
		return artf.Name
	}
	return locale.Tl("alternative_unknown_pkg_name", "unknown")
}

func (s *Setup) DownloadsFromBuild(buildStatus *headchef_models.BuildStatusResponse, storedArtifacts store.StoredArtifactMap) ([]artifact.ArtifactDownload, error) {
	downloads, err := artifact.NewDownloadsFromBuild(buildStatus)
	if err != nil {
		return nil, errs.Wrap(err, "Failed to extract downloads from build result.")
	}
	var res []artifact.ArtifactDownload
	for _, d := range downloads {
		if _, ok := storedArtifacts[d.ArtifactID]; ok {
			continue
		}
		res = append(res, d)
	}
	return res, nil
}
