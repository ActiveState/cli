package runtime

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
)

// installActivePerl will unpack the installer archive, locate the install script, and then use the installer
// script to install an ActivePerl runtime to the configured runtime dir. Any failures
// during this process will result in a failed installation and the install-dir being removed.
func (installer *Installer) installActivePerl(archivePath string, installDir string) *failures.Failure {
	prefix, fail := installer.extractPerlRelocationPrefix(installDir)
	if fail != nil {
		return fail
	}

	// relocate perl
	return installer.Relocate(prefix, installDir)
}

// extractPerlRelocationPrefix will extract the prefix that needs to be replaced for this installation.
func (installer *Installer) extractPerlRelocationPrefix(installDir string) (string, *failures.Failure) {
	relocFile := filepath.Join(installDir, "bin", "reloc_perl")
	if !fileutils.FileExists(relocFile) {
		return "", FailRuntimeNoExecutable.New("installer_err_runtime_no_file", installDir, "bin/reloc_perl")
	}

	f, err := os.Open(relocFile)
	if err != nil {
		return "", failures.FailIO.Wrap(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Scan()
	line := scanner.Text()

	rx := regexp.MustCompile(`#!(.*)/bin`)
	match := rx.FindStringSubmatch(line)
	if len(match) != 2 {
		return "", FailRuntimeNoPrefixes.New("installer_err_fail_obtain_prefixes", installDir)
	}

	return match[1], nil
}
