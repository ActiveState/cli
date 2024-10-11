//go:build !darwin
// +build !darwin

package camel

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
)

// installActivePerl will unpack the installer archive, locate the install script, and then use the installer
// script to install an ActivePerl runtime to the configured runtime dir. Any failures
// during this process will result in a failed installation and the install-dir being removed.
func (m *metaData) perlRelocationDir(installRoot string) (string, error) {
	relocFile := filepath.Join("bin", "reloc_perl")
	if runtime.GOOS == "windows" {
		relocFile = filepath.Join("bin", "config_data")
	}
	relocFilePath := filepath.Join(installRoot, relocFile)
	if !fileutils.FileExists(relocFilePath) {
		return "", locale.NewError("installer_err_runtime_no_file", "", installRoot, relocFile)
	}

	f, err := os.Open(relocFilePath)
	if err != nil {
		return "", errs.Wrap(err, "Open %s failed", relocFilePath)
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
		return "", errs.Wrap(err, "Failed to parse relocation script, could not match '%s' against '%s'", rx.String(), line)
	}

	return match[1], nil
}
