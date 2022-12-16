package updater

import (
	"github.com/ActiveState/cli/internal-as/errs"
	"github.com/ActiveState/cli/internal-as/osutils"
	"github.com/ActiveState/cli/internal-as/unarchiver"
	"github.com/ActiveState/cli/internal/installation"
)

func blobUnarchiver(blob []byte) *unarchiver.ZipBlob {
	return unarchiver.NewZipBlob(blob)
}

func checkAdmin() error {
	installContext, err := installation.GetContext()
	if err != nil {
		return errs.Wrap(err, "Could not get initial installation context")
	}

	isAdmin, err := osutils.IsAdmin()
	if err != nil {
		return errs.Wrap(err, "Could not determine if current user is admin")
	}

	if installContext.InstalledAsAdmin && !isAdmin {
		return errPrivilegeMistmatch
	}

	return nil
}
