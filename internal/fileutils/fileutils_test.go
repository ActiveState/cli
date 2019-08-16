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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ActiveState/cli/internal/environment"
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
	err = ReplaceAll(tmpfile, "%%FIND%%", "REPL", func(string, []byte) bool { return true })
	assert.NoError(t, err, "Replacement worked")

	// Verify the file size changed for text files.
	oldFileStat, err := os.Stat(testfile)
	assert.NoError(t, err, "Can read attributes of test file")
	newFileStat, err := os.Stat(tmpfile)
	assert.NoError(t, err, "Can read attributes of replacement file")
	assert.True(t, oldFileStat.Size() > newFileStat.Size(), "Replacement file is smaller, actual old: %d, vs new: %d", oldFileStat.Size(), newFileStat.Size())

	// Compare the orig test.go file with the new one.
	oldBytes, err := ioutil.ReadFile(testfile)
	assert.NoError(t, err, "Read original text file")
	newBytes, err := ioutil.ReadFile(tmpfile)
	assert.NoError(t, err, "Read new text file")
	assert.NotEqual(t, string(oldBytes), string(newBytes), "Copy succeeded")
	assert.Equal(t, string(bytes.Replace(oldBytes, []byte("%%FIND%%"), []byte("REPL"), -1)), string(newBytes), "Copy succeeded")
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
	err = ReplaceAll(exe, "%%FIND%%", "REPLTOOLONG", func(string, []byte) bool { return true })
	assert.Error(t, err, "Replacement text was too long")
	err = ReplaceAll(exe, "%%FIND%%", "REPL", func(string, []byte) bool { return true })
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
	require.NoError(t, failure.ToError())
	assert.True(t, isEmpty)
}

func TestEmptyDir_HasRegularFile(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test-dir-has-file")
	require.NoError(t, err)

	f, failure := Touch(path.Join(tmpdir, "regular-file"))
	require.NoError(t, failure.ToError())
	defer os.Remove(f.Name())

	isEmpty, failure := IsEmptyDir(tmpdir)
	require.NoError(t, failure.ToError())
	assert.False(t, isEmpty)
}

func TestEmptyDir_HasSubDir(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "test-dir-has-dir")
	require.NoError(t, err)

	require.Nil(t, Mkdir(path.Join(tmpdir, "some-dir")))

	isEmpty, failure := IsEmptyDir(tmpdir)
	require.NoError(t, failure.ToError())
	assert.False(t, isEmpty)
}

func getTempDir(t *testing.T, appendStr string) string {
	dir := "test-dir"
	if appendStr == "" {
		dir = "test-dir-" + appendStr
	}
	tmpdir, err := ioutil.TempDir("", dir)
	require.NoError(t, err)
	return tmpdir
}

func TestAmendFile_BadArg(t *testing.T) {
	path := path.Join(getTempDir(t, "bad-args"), "file.txt")

	// Due to the type def we don't need to test - ints
	// fails as an overflow before you can even run your code.
	fail := AmendFile(path, []byte(""), 99)
	assert.Error(t, fail.ToError(), "Reject bad flag")
	assert.False(t, FileExists(path), "No file should be created.")
}

func TestAppend(t *testing.T) {
	path := path.Join(getTempDir(t, "append-file"), "file.txt")

	fail := WriteFile(path, []byte("a"))
	require.NoError(t, fail.ToError())

	// Append
	fail = AmendFile(path, []byte("a"), AmendByAppend)
	assert.NoError(t, fail.ToError(), "Should be able to write to empty file.")

	fail = AppendToFile(path, []byte("b"))
	assert.NoError(t, fail.ToError(), "Should be able to append to file.")

	assert.Equal(t, []byte("aab"), ReadFileUnsafe(path))
}

func TestWriteFile(t *testing.T) {
	file, err := ioutil.TempFile("", "cli-test-writefile-replace")
	require.NoError(t, err)
	file.Close()

	// Set file read-only to test if chmodding from WriteFile works
	os.Chmod(file.Name(), 0444)

	fail := WriteFile(file.Name(), []byte("abc"))
	require.NoError(t, fail.ToError())

	fail = WriteFile(file.Name(), []byte("def"))
	require.NoError(t, fail.ToError())

	assert.Equal(t, "def", string(ReadFileUnsafe(file.Name())))
}

func TestWriteFile_Prepend(t *testing.T) {
	path := path.Join(getTempDir(t, "prepend-file"), "file.txt")

	fail := WriteFile(path, []byte("a"))
	require.NoError(t, fail.ToError())

	// Prepend
	fail = AmendFile(path, []byte("b"), AmendByPrepend)
	assert.NoError(t, fail.ToError(), "Should be able to write to empty file.")

	fail = PrependToFile(path, []byte("a"))
	assert.NoError(t, fail.ToError(), "Should be able to prepend to file.")

	assert.Equal(t, []byte("aba"), ReadFileUnsafe(path))
}

func TestWriteFile_OverWrite(t *testing.T) {
	path := path.Join(getTempDir(t, "overwrite-file"), "file.txt")

	// Overwrite
	fail := WriteFile(path, []byte("cba"))
	assert.NoError(t, fail.ToError(), "Should be able to write to empty file.")

	fail = WriteFile(path, []byte("abc"))
	assert.NoError(t, fail.ToError(), "Should be able to overwrite file.")

	assert.Equal(t, []byte("abc"), ReadFileUnsafe(path), "Should have overwritten file")
}

func TestTouch(t *testing.T) {
	dir := getTempDir(t, "touch-file")
	noParentPath := path.Join(dir, "randocalrizian", "file.txt")
	path := path.Join(dir, "file.txt")

	{
		file, fail := Touch(path)
		require.NoError(t, fail.ToError(), "File created without fail")
		file.Close()
	}

	{
		file, fail := Touch(noParentPath)
		require.NoError(t, fail.ToError(), "File with missing parent created without fail")
		file.Close()
	}
}

func TestReadFile(t *testing.T) {
	path := path.Join(getTempDir(t, "read-file"), "file.txt")

	_, fail := ReadFile(path)
	assert.Error(t, fail.ToError(), "File doesn't exist, fail.")

	content := []byte("pizza time")
	fail = WriteFile(path, content)
	assert.NoError(t, fail.ToError(), "File write without fail")

	var b []byte
	b, fail = ReadFile(path)
	assert.NoError(t, fail.ToError(), "File doesn't exist, fail.")
	assert.Equal(t, content, b, "Content should be the same")
}

func TestExecutable(t *testing.T) {
	assert.True(t, IsExecutable(os.Args[0]), "Can detect that file is executable")
}

func TestCreateTempExecutable(t *testing.T) {
	patPrefix := "abc"
	patSuffix := ".xxx"
	pattern := patPrefix + "*" + patSuffix
	data := []byte("this is a test")

	name, fail := WriteTempFile("", pattern, data, 0700)
	require.NoError(t, fail.ToError())
	require.FileExists(t, name)
	defer os.Remove(name)

	assert.True(t, len(name) > len(pattern))
	assert.Contains(t, name, patPrefix)
	assert.Contains(t, name, patSuffix)

	info, err := os.Stat(name)
	require.NoError(t, err)
	assert.True(t, info.Size() > 0)

	res := int64(0500 & info.Mode()) // readable/executable by user
	if runtime.GOOS == "windows" {
		res = int64(0400 & info.Mode()) // readable by user
	}
	assert.NotZero(t, res, "file should be readable/executable")
}
