package environment

// This package may NOT depend on failures (directly or indirectly)

import (
	"errors"
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// GetRootPath returns the root path of the library we're under
func GetRootPath() (string, error) {
	pathsep := string(os.PathSeparator)

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("Could not call Caller(0)")
	}

	fmt.Println("Caller file:", file)
	abs := filepath.Dir(file)
	fmt.Println("Absolute path:", abs)

	// If we're receiving a relative path resolve it to absolute
	if abs[0:1] != "/" && abs[1:2] != ":" {
		gopath := os.Getenv("GOPATH")
		if gopath == "" {
			gopath = build.Default.GOPATH
		}
		abs = filepath.Join(gopath, "src", abs)
	}

	// When tests are ran with coverage the location of this file is changed to a temp file, and we have to
	// adjust accordingly
	if strings.HasSuffix(abs, "_obj_test") {
		abs = ""
	}

	// If we're in a temp _obj we need to account for it in the path
	if strings.HasSuffix(abs, "_obj") {
		abs = filepath.Join(abs, "..")
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
