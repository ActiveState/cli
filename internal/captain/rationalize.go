package captain

import (
	"errors"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/prompt"
	"github.com/ActiveState/cli/internal/runbits/rationalize"
	"github.com/ActiveState/cli/pkg/buildscript"
	"github.com/ActiveState/cli/pkg/localcommit"
)

func rationalizeError(err *error) {
	var errInvalidCommitID *localcommit.ErrInvalidCommitID

	switch {
	case err == nil:
		return

	// Do not modify an existing user-facing error.
	case errs.IsUserFacing(*err):
		return

	// Project not found.
	case errors.Is(*err, rationalize.ErrNoProject):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_no_project"),
			errs.SetInput())

	// Invalid commit ID.
	case errors.As(*err, &errInvalidCommitID):
		*err = errs.WrapUserFacing(*err,
			locale.Tr("err_commit_id_invalid", errInvalidCommitID.CommitID),
			errs.SetInput())

	// Outdated build script.
	case errors.Is(*err, buildscript.ErrOutdatedAtTime):
		*err = errs.WrapUserFacing(*err,
			locale.T("err_outdated_buildscript"),
			errs.SetInput())

	case errors.Is(*err, prompt.ErrNoForceOption):
		*err = errs.WrapUserFacing(*err,
			locale.T("err_prompt_no_force_option",
				"This command has a prompt that does not support the '[ACTIONABLE]--force[/RESET]' flag."))
	}
}
