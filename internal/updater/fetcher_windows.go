package updater

import "github.com/ActiveState/cli/internal/unarchiver"

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
