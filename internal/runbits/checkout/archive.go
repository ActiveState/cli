package checkout

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/go-openapi/strfmt"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/internal/unarchiver"
	"github.com/ActiveState/cli/pkg/buildplan"
	"github.com/ActiveState/cli/pkg/project"
)

type Archive struct {
	Dir        string
	Namespace  *project.Namespaced
	Branch     string
	PlatformID strfmt.UUID
	BuildPlan  *buildplan.BuildPlan
}

const ArchiveExt = ".tar.gz"
const ArtifactExt = ".tar.gz"

type projectJson struct {
	Owner      string `json:"org_name"`
	Project    string `json:"project_name"`
	Branch     string `json:"branch"`
	CommitID   string `json:"commit_id"`
	PlatformID string `json:"platform_id"`
}

// NewArchive unpacks the given archive to a temporary location.
// The caller should invoke the `Cleanup()` method when finished with this archive.
func NewArchive(archivePath string) (_ *Archive, rerr error) {
	dir, err := os.MkdirTemp("", "")
	if err != nil {
		return nil, errs.Wrap(err, "Unable to create temporary directory")
	}
	defer func() {
		if rerr == nil {
			return
		}
		// Delete the temporary directory if there was an error unpacking the archive.
		if err := os.RemoveAll(dir); err != nil {
			if rerr != nil {
				err = errs.Pack(rerr, errs.Wrap(err, "Unable to delete temporary directory"))
			}
			rerr = err
		}
	}()

	// Prepare.
	ua := unarchiver.NewTarGz()
	f, size, err := ua.PrepareUnpacking(archivePath, dir)
	if err != nil {
		if err2 := os.RemoveAll(dir); err2 != nil {
			err = errs.Pack(err, errs.Wrap(err2, "Unable to delete temporary directory"))
		}
		return nil, errs.Wrap(err, "Unable to read archive")
	}

	// Unpack.
	err = ua.Unarchive(f, size, dir)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to extract archive")
	}

	// Read from project.json.
	ns, branch, platformID, err := readProject(dir)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to read project from archive")
	}

	// Read from buildplan.json.
	buildPlan, err := readBuildPlan(dir)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to read buildplan from archive")
	}

	return &Archive{dir, ns, branch, platformID, buildPlan}, nil
}

// Cleanup should be called after the archive is no longer needed.
// Otherwise, its contents will remain on disk.
func (a *Archive) Cleanup() error {
	return os.RemoveAll(a.Dir)
}

// readProject reads and returns a project namespace (with commitID) and branch from
// "project.json", as well as a platformID.
func readProject(dir string) (*project.Namespaced, string, strfmt.UUID, error) {
	projectBytes, err := fileutils.ReadFile(filepath.Join(dir, "project.json"))
	if err != nil {
		return nil, "", "", errs.Wrap(err, "Invalid archive: project.json not found")
	}

	var proj *projectJson
	err = json.Unmarshal(projectBytes, &proj)
	if err != nil {
		return nil, "", "", errs.Wrap(err, "Unable to read project.json")
	}

	ns := &project.Namespaced{Owner: proj.Owner, Project: proj.Project, CommitID: ptr.To(strfmt.UUID(proj.CommitID))}
	return ns, proj.Branch, strfmt.UUID(proj.PlatformID), nil
}

// readBuildPlan reads and returns a buildplan from "buildplan.json".
func readBuildPlan(dir string) (*buildplan.BuildPlan, error) {
	buildplanBytes, err := fileutils.ReadFile(filepath.Join(dir, "buildplan.json"))
	if err != nil {
		return nil, errs.Wrap(err, "Invalid archive: buildplan.json not found")
	}

	return buildplan.Unmarshal(buildplanBytes)
}
