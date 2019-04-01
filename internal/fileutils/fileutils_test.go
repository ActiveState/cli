package fileutils

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Copies the file associated with the given filename to a temp dir and returns
// the path to the temp file. The temp file and its directory must be manually
// removed.
func copyFileToTempDir(t *testing.T, filename string) string {
	fileBytes, err := ioutil.ReadFile(filename)
	assert.NoError(t, err, "Read file to copy")

	tmpdir, err := ioutil.TempDir("", "activestatecli-test")
	assert.NoError(t, err, "Created a temp dir")

	tmpfile := filepath.Join(tmpdir, filepath.Base(filename))
	err = ioutil.WriteFile(tmpfile, fileBytes, 0666)
	assert.NoError(t, err, "Wrote to temp file")

	return tmpfile
}

func TestReplaceAllTextFile(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")

	// Copy test.go to temp dir.
	testfile := filepath.Join(root, "internal", "fileutils", "testdata", "test.go")
	tmpfile := copyFileToTempDir(t, testfile)
	defer os.RemoveAll(filepath.Dir(tmpfile))

	// Perform the replacement.
	err = ReplaceAll(tmpfile, "%%FIND%%", "REPL")
	assert.NoError(t, err, "Replacement worked")

	// Verify the file size changed for text files.
	oldFileStat, err := os.Stat(testfile)
	assert.NoError(t, err, "Can read attributes of test file")
	newFileStat, err := os.Stat(tmpfile)
	assert.NoError(t, err, "Can read attributes of replacement file")
	assert.True(t, oldFileStat.Size() > newFileStat.Size(), "Replacement file is smaller")

	// Compare the orig test.go file with the new one.
	oldBytes, err := ioutil.ReadFile(testfile)
	assert.NoError(t, err, "Read original text file")
	newBytes, err := ioutil.ReadFile(tmpfile)
	assert.NoError(t, err, "Read new text file")
	assert.Equal(t, bytes.Replace(oldBytes, []byte("%%FIND%%"), []byte("REPL"), -1), newBytes, "Copy succeeded")
}

func TestReplaceAllExe(t *testing.T) {
	gobin := "go"
	goroot := os.Getenv("GOROOT")
	if goroot != "" {
		gobin = filepath.Join(goroot, "bin", "go")
	}

	root, err := environment.GetRootPath()
	assert.NoError(t, err, "Should detect root path")
	cwd, err := os.Getwd()
	assert.NoError(t, err, "Determined working directory")

	// Copy test.go to temp dir.
	tmpfile := copyFileToTempDir(t, filepath.Join(root, "internal", "fileutils", "testdata", "test.go"))
	defer os.RemoveAll(filepath.Dir(tmpfile))

	// Build test.go in the temp dir.
	err = os.Chdir(filepath.Dir(tmpfile))
	defer os.Chdir(cwd) // restore
	assert.NoError(t, err, "Changed to the tempfile's directory")
	exe := "test"
	if runtime.GOOS == "windows" {
		exe += ".exe"
	}
	cmd := exec.Command(gobin, "build", "-o", exe, filepath.Base(tmpfile))
	err = cmd.Run()
	assert.NoError(t, err, "Ran go build")
	oldExeStat, err := os.Stat(filepath.Join(filepath.Dir(tmpfile), exe))
	assert.NoError(t, err, "Can read attributes of exe")
	oldExeSize := oldExeStat.Size() // read now since exe will be replaced

	// Run the exe and fetch original output.
	sep := ":"
	if runtime.GOOS == "windows" {
		sep = ";"
	}
	path := os.Getenv("PATH")
	os.Setenv("PATH", fmt.Sprintf("%s%s%s", filepath.Dir(tmpfile), sep, path))
	defer os.Setenv("PATH", path) // restore
	cmd = exec.Command(exe)
	oldOutput, err := cmd.Output()
	assert.NoError(t, err, "Go exe ran")
	assert.True(t, len(oldOutput) > 0, "Stdout read")

	// Perform binary replace.
	err = ReplaceAll(exe, "%%FIND%%", "REPLTOOLONG")
	assert.Error(t, err, "Replacement text was too long")
	err = ReplaceAll(exe, "%%FIND%%", "REPL")
	assert.NoError(t, err, "Replacement worked")

	// Verify the file size is identical for binary files.
	newExeStat, err := os.Stat(filepath.Join(filepath.Dir(tmpfile), exe))
	assert.NoError(t, err, "Can read attributes of replacement exe")
	assert.True(t, oldExeSize == newExeStat.Size(), "Replacement exe is same size")

	// Run the replacement exe and fetch new output.
	// Note: executables produced by Go appear to encode string length somewhere,
	// rather than terminate strings with NUL bytes like C/C++-compiled
	// executables. Account for that.
	cmd = exec.Command(exe)
	newOutput, err := cmd.Output()
	assert.NoError(t, err, "Replacement exe ran")
	assert.Equal(t, bytes.Replace(oldOutput, []byte("%%FIND%%"), []byte("REPL\x00\x00\x00\x00"), -1), newOutput, "Copy succeeded")
}

func TestEmptyDir_IsEmpty(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test-dir-is-empty")
	require.NoError(t, err)

	isEmpty, failure := IsEmptyDir(tmpdir)
	require.Nil(t, failure)
	assert.True(t, isEmpty)
}

func TestEmptyDir_HasRegularFile(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test-dir-has-file")
	require.NoError(t, err)

	f, failure := Touch(path.Join(tmpdir, "regular-file"))
	require.Nil(t, failure)
	defer os.Remove(f.Name())

	isEmpty, failure := IsEmptyDir(tmpdir)
	require.Nil(t, failure)
	assert.False(t, isEmpty)
}

func TestEmptyDir_HasSubDir(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test-dir-has-dir")
	require.NoError(t, err)

	require.Nil(t, Mkdir(path.Join(tmpdir, "some-dir")))

	isEmpty, failure := IsEmptyDir(tmpdir)
	require.Nil(t, failure)
	assert.False(t, isEmpty)
}

func TestWriteFile_BadArg(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test-dir-write-file")
	require.NoError(t, err)
	path := path.Join(tmpdir, "file.txt")

	// Due to the type def we don't need to test - ints
	// fails as an overflow before you can even run your code.
	fail := WriteFile(path, "", 3)
	assert.NotNil(t, fail, "Reject bad flag")
	assert.False(t, FileExists(path), "No file should be created.")
}

func TestTouchFile_Append(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test-dir-write-file")
	require.NoError(t, err)
	path := path.Join(tmpdir, "file.txt")

	// Append
	err = WriteFile(path, "a", OverwriteFile)
	assert.Nil(t, err, "Should be able to write to empty file.")
	err = WriteFile(path, "b", AppendToFile)
	assert.Nil(t, err, "Should be able to append to file.")
	assert.Equal(t, []byte("ab"), ReadFileUnsafe(path), "Should be equal")
}

func TestTouchFile_Prepend(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test-dir-write-file")
	require.NoError(t, err)
	path := path.Join(tmpdir, "file.txt")

	// Prepend
	err = WriteFile(path, "b", OverwriteFile)
	assert.Nil(t, err, "Should be able to write to empty file.")
	err = WriteFile(path, "a", PrependToFile)
	assert.Nil(t, err, "Should be able to prepend to file.")
	assert.Equal(t, []byte("ab"), ReadFileUnsafe(path), "Should be equal")
}

func TestTouchFile_OverWrite(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test-dir-write-file")
	require.NoError(t, err)
	path := path.Join(tmpdir, "file.txt")

	// Overwrite
	err = WriteFile(path, "cba", OverwriteFile)
	assert.Nil(t, err, "Should be able to write to empty file.")
	err = WriteFile(path, "abc", OverwriteFile)
	assert.Nil(t, err, "Should be able to overwrite file.")
	assert.Equal(t, []byte("abc"), ReadFileUnsafe(path), "Should have overwritten file")
}

func TestTouchFile(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test-dir-touch-file")
	require.NoError(t, err)
	noParentPath := path.Join(tmpdir, "randocalrizian", "file.txt")
	path := path.Join(tmpdir, "file.txt")

	{
		fail := TouchFile(path)
		assert.Nil(t, fail, "File created without fail")
	}

	{
		fail := TouchFile(noParentPath)
		assert.Nil(t, fail, "File with missing parent created without fail")
	}
}

func TestReadFile(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test-dir-read-file")
	require.NoError(t, err)
	path := path.Join(tmpdir, "file.txt")

	_, fail := ReadFile(path)
	assert.NotNil(t, fail, "File doesn't exist, fail.")

	content := "pizza time"
	fail = WriteFile(path, content, OverwriteFile)
	assert.Nil(t, fail, "File write without fail")

	fail = nil
	var b []byte
	b, fail = ReadFile(path)
	assert.Nil(t, fail, "File doesn't exist, fail.")
	assert.Equal(t, content, string(b), "Content should be the same")
}
