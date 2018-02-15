package environment

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ActiveState/ActiveState-CLI/internal/failures"
)

// GetRootPath returns the root path of the library we're under
func GetRootPath() (string, error) {
	pathsep := string(os.PathSeparator)

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", failures.App.New("Could not call Caller(0)")
	}

	abs := filepath.Dir(file)

	// When tests are ran with coverage the location of this file is changed to a temp file, and we have to
	// adjust accordingly
	if strings.HasSuffix(abs, "_obj_test") {
		abs = ""
	}

	var err error
	abs, err = filepath.Abs(filepath.Join(abs, "..", ".."))

	if err != nil {
		return "", err
	}

	return abs + pathsep, nil
}

// GetRootPathUnsafe returns the root path or panics if it cannot be found (hence the unsafe)
func GetRootPathUnsafe() string {
	path, err := GetRootPath()
	if err != nil {
		panic(err)
	}
	return path
}
