package runtime

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/logging"
)

// installActivePerl will unpack the installer archive, locate the install script, and then use the installer
// script to install an ActivePerl runtime to the configured runtime dir. Any failures
// during this process will result in a failed installation and the install-dir being removed.
func (m *MetaData) perlRelocationDir() (string, error) {
	relocFile := filepath.Join("bin", "reloc_perl")
	if runtime.GOOS == "windows" {
		relocFile = filepath.Join("bin", "config_data")
	}
	relocFilePath := filepath.Join(m.Path, relocFile)
	if !fileutils.FileExists(relocFilePath) {
		return "", FailRuntimeNoExecutable.New("installer_err_runtime_no_file", m.Path, relocFile)
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
		return "", FailRuntimeNoPrefixes.New("installer_err_fail_obtain_prefixes", m.Path)
	}

	return match[1], nil
}

func loadRelocationFile(relocFilePath string) map[string]bool {
	relocBytes, err := ioutil.ReadFile(relocFilePath)
	if err != nil {
		logging.Debug("Could not open relocation file: %v", err)
		return nil
	}
	reloc := string(relocBytes)
	relocMap := map[string]bool{}
	entries := strings.Split(reloc, "\n")
	for _, entry := range entries {
		if entry == "" {
			continue
		}
		info := strings.Split(entry, " ")
		// Place path suffix into map
		relocMap[info[1]] = true
	}
	return relocMap
}
