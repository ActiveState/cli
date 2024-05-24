package envdef

import (
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
)

// Constants is a map of constants that are being expanded in environment variables and file transformations to their installation-specific values
type Constants map[string]string

// NewConstants initializes a new map of constants that will need to be set to installation-specific values
// Currently it only has one field `INSTALLDIR`
func NewConstants(installdir string) (Constants, error) {
	dir, err := fileutils.CaseSensitivePath(installdir)
	if err != nil {
		return nil, errs.Wrap(err, "Could not search for case sensitive install dir")
	}

	return map[string]string{
		`INSTALLDIR`: dir,
	}, nil
}
