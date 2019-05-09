package runtime

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"

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
	relocFile := filepath.Join("bin", "reloc_perl")
	if runtime.GOOS == "windows" {
		relocFile = filepath.Join("bin", "config_data")
	}
	relocFilePath := filepath.Join(installDir, relocFile)
	if !fileutils.FileExists(relocFilePath) {
		return "", FailRuntimeNoExecutable.New("installer_err_runtime_no_file", installDir, relocFile)
	}

	f, err := os.Open(relocFilePath)
	if err != nil {
		return "", failures.FailIO.Wrap(err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Scan()
	line := scanner.Text()

	// Can't use filepath.Separator because we need to escape the backslash on Windows
	separator := `/`
	if runtime.GOOS == "windows" {
		separator = `\\`
	}

	rx := regexp.MustCompile(fmt.Sprintf(`#!(.*)%sbin`, separator))
	match := rx.FindStringSubmatch(line)
	if len(match) != 2 {
		return "", FailRuntimeNoPrefixes.New("installer_err_fail_obtain_prefixes", installDir)
	}

	return match[1], nil
}
