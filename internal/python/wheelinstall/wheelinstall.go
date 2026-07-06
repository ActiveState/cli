// Package wheelinstall installs a pure-Python wheel into a site-packages
// directory.
package wheelinstall

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/unarchiver"
)

// installerName is the contents of the .dist-info/INSTALLER file (PEP 376).
const installerName = "state-tool"

// Install extracts the wheel at wheelPath into sitePackagesDir and writes the
// INSTALLER marker into its .dist-info. Entries that would escape sitePackagesDir,
// and wheels without a .dist-info, are rejected.
func Install(wheelPath, sitePackagesDir string) error {
	if err := fileutils.MkdirUnlessExists(sitePackagesDir); err != nil {
		return errs.Wrap(err, "could not create site-packages directory")
	}

	wheel, err := os.Open(wheelPath)
	if err != nil {
		return errs.Wrap(err, "could not open wheel")
	}
	defer wheel.Close()

	// A wheel is a zip; untrusted-source mode confines entries to the destination.
	ua := unarchiver.NewZip(unarchiver.WithUntrustedSource())
	if err := ua.Unarchive(wheel, sitePackagesDir); err != nil {
		return errs.Wrap(err, "could not extract wheel")
	}

	if err := writeInstaller(sitePackagesDir); err != nil {
		return errs.Wrap(err, "could not record installer")
	}
	return nil
}

// writeInstaller writes the INSTALLER marker into the wheel's sole *.dist-info
// directory, erroring if there is none.
func writeInstaller(sitePackagesDir string) error {
	entries, err := os.ReadDir(sitePackagesDir)
	if err != nil {
		return errs.Wrap(err, "could not read site-packages directory")
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasSuffix(e.Name(), ".dist-info") {
			marker := filepath.Join(sitePackagesDir, e.Name(), "INSTALLER")
			if err := fileutils.WriteFile(marker, []byte(installerName+"\n")); err != nil {
				return errs.Wrap(err, "could not write INSTALLER")
			}
			return nil
		}
	}
	return errs.New("wheel has no .dist-info directory")
}
