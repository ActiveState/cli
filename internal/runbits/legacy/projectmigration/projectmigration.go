package projectmigration

import (
	_ "embed"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/rtutils/ptr"
	"github.com/ActiveState/cli/pkg/project"
	"github.com/go-openapi/strfmt"
)

type projecter interface {
	Dir() string
	LegacyCommitID() string
	SetLegacyCommit(commitID string) error
}

type migrator struct {
	out  output.Outputer
	proj projecter
}

func New(out output.Outputer, pj projecter) *migrator {
	return &migrator{out: out, proj: pj}
}

// setupProject will ensure that the stored project matches the path that was requested. In most cases this should match,
// and setting up a new project instance will be rare.
func (m *migrator) setupProject(pjpath string) error {
	if !ptr.IsNil(m.proj) && m.proj.Dir() == pjpath {
		return nil
	}
	var err error
	m.proj, err = project.FromPath(pjpath)
	return err
}

// Migrate returns the legacy commit ID and updates the activestate.yaml with instructions on dropping the legacy commit.
func (m *migrator) Migrate(pjpath string) (strfmt.UUID, error) {
	logging.Debug("Migrating project to new localcommit format: %s", pjpath)
	if err := m.setupProject(pjpath); err != nil {
		return "", err
	}

	configPath := filepath.Join(m.proj.Dir(), constants.ConfigFileName)

	// Add comment to activestate.yaml explaining migration
	asB, err := fileutils.ReadFile(configPath)
	if err != nil {
		return "", errs.Wrap(err, "Could not read activestate.yaml")
	}

	asB = append([]byte(locale.T("projectmigration_asyaml_comment")), asB...)
	if err := fileutils.WriteFile(configPath, asB); err != nil {
		return "", errs.Wrap(err, "Could not write to activestate.yaml")
	}

	if !strfmt.IsUUID(m.proj.LegacyCommitID()) {
		return "", locale.NewInputError("err_commit_id_invalid", m.proj.LegacyCommitID())
	}

	return strfmt.UUID(m.proj.LegacyCommitID()), nil
}

func (m *migrator) Set(pjpath string, commitID string) error {
	if err := m.setupProject(pjpath); err != nil {
		return err
	}

	if m.proj.LegacyCommitID() != "" {
		if err := m.proj.SetLegacyCommit(commitID); err != nil {
			return errs.Wrap(err, "Could not set legacy commit")
		}
	}
	return nil
}
