package osutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ActiveState/cli/internal/fileutils"
)

// CreateConfigFile will create a file in the config dir with the given file name.
func CreateConfigFile(configPath string, fileName string, fileMode os.FileMode) (*os.File, error) {
	filename := filepath.Join(configPath, fileName)
	return os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileMode)
}

// ReadConfigFile will read the contents of a file in the config dir.
func ReadConfigFile(configPath, fileName string) (string, error) {
	contents, err := ioutil.ReadFile(filepath.Join(configPath, fileName))
	return string(contents), err
}

// RemoveConfigFile will remove a file from the config dir with the given file name.
func RemoveConfigFile(configPath, fileName string) error {
	return os.Remove(filepath.Join(configPath, fileName))
}

// StatConfigFile returns the os.FileInfo for a file in the config dir.
func StatConfigFile(configPath, fileName string) (os.FileInfo, error) {
	return os.Stat(filepath.Join(configPath, fileName))
}

// GetTestFile returns the path to the given fileName in the calling function's `testdata` directory.
func GetTestFile(fileName string) string {
	callerPath := getCallerPath()
	return filepath.Join(callerPath, "testdata", fileName)
}

// ReadTestFile will read the contents of a file from the `testdata` directory relative to the
// path of the calling function file. This function assumes it is called directly from a function
// in a file in the directory the `testdata` exists in.
func ReadTestFile(fileName string) (string, error) {
	contents, err := ioutil.ReadFile(GetTestFile(fileName))
	return string(contents), err
}

// CopyTestFileToConfigDir copies a file in a relatve `testdata` dir to the caller of this function
// to the config dir as some target filename with some target FileMode.
func CopyTestFileToConfigDir(configPath, testFileName, targetFileName string, targetFileMode os.FileMode) error {
	testFileContent, err := ReadTestFile(testFileName)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(configPath, targetFileName), []byte(testFileContent), targetFileMode)
}

// getCallerPath returns the filesystem path of the caller to this func so long as it's not
// in this file's directory.
func getCallerPath() string {
	_, currentFile, _, ok := runtime.Caller(0)
	file := currentFile
	skip := 1 // skip position

	// find the file of the previous caller that is not in this file
	for file == currentFile && ok {
		_, file, _, ok = runtime.Caller(skip)
		skip++
	}

	if file == "" || file == currentFile {
		panic("Could not get caller")
	}

	return filepath.Dir(file)
}

// PrepareDir prepares a path for use in tests (ensures it exists and ensures the path is concistent)
func PrepareDir(path string) string {
	if path == "" {
		return path
	}

	var err error
	path, err = fileutils.PrepareDir(path)
	if err != nil {
		panic(fmt.Sprintf("PrepareDir error: %v", err))
	}

	return path
}
