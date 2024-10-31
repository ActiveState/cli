package fileutils

import (
	"bytes"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/osutils/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thoas/go-funk"
)

// Copies the file associated with the given filename to a temp dir and returns
// the path to the temp file. The temp file and its directory must be manually
// removed.
func copyFileToTempDir(t *testing.T, filename string) string {
	fileBytes, err := os.ReadFile(filename)
	assert.NoError(t, err, "Read file to copy")

	tmpdir, err := os.MkdirTemp("", "activestatecli-test")
	assert.NoError(t, err, "Created a temp dir")

	tmpfile := filepath.Join(tmpdir, filepath.Base(filename))
	err = os.WriteFile(tmpfile, fileBytes, 0666)
	assert.NoError(t, err, "Wrote to temp file")

	return tmpfile
}

func TestFindFileInPath(t *testing.T) {
	root, err := environment.GetRootPath()
	assert.NoError(t, err, "should detect root path")

	expectpath := filepath.Join(root, "internal", "fileutils", "testdata")
	expectfile := filepath.Join(expectpath, "test.go")
	testpath := filepath.Join(expectpath, "extra-dir", "another-dir")
	resultpath, err := FindFileInPath(testpath, "test.go")
	assert.Nilf(t, err, "No failure expected")
	assert.Equal(t, resultpath, expectfile)

	nonExistentPath := filepath.FromSlash("/dir1/dir2")
	_, err = FindFileInPath(nonExistentPath, "FILE_THAT_SHOULD_NOT_EXIST")

	assert.ErrorIs(t, err, ErrorFileNotFound)
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
	assert.True(t, oldFileStat.Size() > newFileStat.Size(), "Replacement file is smaller, actual old: %d, vs new: %d", oldFileStat.Size(), newFileStat.Size())

	// Compare the orig test.go file with the new one.
	oldBytes, err := os.ReadFile(testfile)
	assert.NoError(t, err, "Read original text file")
	newBytes, err := os.ReadFile(tmpfile)
	assert.NoError(t, err, "Read new text file")
	assert.NotEqual(t, string(oldBytes), string(newBytes), "Copy succeeded")
	assert.Equal(t, string(bytes.Replace(oldBytes, []byte("%%FIND%%"), []byte("REPL"), -1)), string(newBytes), "Copy succeeded")
}

func TestEmptyDir_IsEmpty(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "test-dir-is-empty")
	require.NoError(t, err)

	isEmpty, err := IsEmptyDir(tmpdir)
	require.NoError(t, err)
	assert.True(t, isEmpty)
}

func TestEmptyDir_HasRegularFile(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "test-dir-has-file")
	require.NoError(t, err)

	err = Touch(path.Join(tmpdir, "regular-file"))
	require.NoError(t, err)

	isEmpty, err := IsEmptyDir(tmpdir)
	require.NoError(t, err)
	assert.False(t, isEmpty)
}

func TestEmptyDir_HasSubDir(t *testing.T) {
	tmpdir, err := os.MkdirTemp("", "test-dir-has-dir")
	require.NoError(t, err)

	require.Nil(t, Mkdir(path.Join(tmpdir, "some-dir")))

	isEmpty, err := IsEmptyDir(tmpdir)
	require.NoError(t, err)
	assert.False(t, isEmpty)
}

func getTempDir(t *testing.T, appendStr string) string {
	dir := "test-dir"
	if appendStr == "" {
		dir = "test-dir-" + appendStr
	}
	tmpdir, err := os.MkdirTemp("", dir)
	require.NoError(t, err)
	return tmpdir
}

func TestAmendFile_BadArg(t *testing.T) {
	path := path.Join(getTempDir(t, "bad-args"), "file.txt")

	// Due to the type def we don't need to test - ints
	// fails as an overflow before you can even run your code.
	err := AmendFile(path, []byte(""), 99)
	assert.Error(t, err, "Reject bad flag")
	assert.False(t, FileExists(path), "No file should be created.")
}

func TestAppend(t *testing.T) {
	path := path.Join(getTempDir(t, "append-file"), "file.txt")

	err := WriteFile(path, []byte("a"))
	require.NoError(t, err)

	// Append
	err = AmendFile(path, []byte("a"), AmendByAppend)
	assert.NoError(t, err, "Should be able to write to empty file.")

	err = AppendToFile(path, []byte("b"))
	assert.NoError(t, err, "Should be able to append to file.")

	assert.Equal(t, []byte("aab"), ReadFileUnsafe(path))
}

func TestWriteFile(t *testing.T) {
	file, err := os.CreateTemp("", "cli-test-writefile-replace")
	require.NoError(t, err)
	file.Close()

	// Set file read-only to test if chmodding from WriteFile works
	err = os.Chmod(file.Name(), 0444)
	require.NoError(t, err)

	err = WriteFile(file.Name(), []byte("abc"))
	require.NoError(t, err)

	err = WriteFile(file.Name(), []byte("def"))
	require.NoError(t, err)

	assert.Equal(t, "def", string(ReadFileUnsafe(file.Name())))
}

func TestWriteFile_Prepend(t *testing.T) {
	path := path.Join(getTempDir(t, "prepend-file"), "file.txt")

	err := WriteFile(path, []byte("a"))
	require.NoError(t, err)

	// Prepend
	err = AmendFile(path, []byte("b"), AmendByPrepend)
	assert.NoError(t, err, "Should be able to write to empty file.")

	err = PrependToFile(path, []byte("a"))
	assert.NoError(t, err, "Should be able to prepend to file.")

	assert.Equal(t, []byte("aba"), ReadFileUnsafe(path))
}

func TestWriteFile_OverWrite(t *testing.T) {
	path := path.Join(getTempDir(t, "overwrite-file"), "file.txt")

	// Overwrite
	err := WriteFile(path, []byte("cba"))
	assert.NoError(t, err, "Should be able to write to empty file.")

	err = WriteFile(path, []byte("abc"))
	assert.NoError(t, err, "Should be able to overwrite file.")

	assert.Equal(t, []byte("abc"), ReadFileUnsafe(path), "Should have overwritten file")
}

func TestTouch(t *testing.T) {
	dir := getTempDir(t, "touch-file")
	noParentPath := path.Join(dir, "randocalrizian", "file.txt")
	path := path.Join(dir, "file.txt")

	{
		err := Touch(path)
		require.NoError(t, err, "File created without fail")
	}

	{
		err := Touch(noParentPath)
		require.NoError(t, err, "File with missing parent created without fail")
	}
}

func TestReadFile(t *testing.T) {
	path := path.Join(getTempDir(t, "read-file"), "file.txt")

	_, err := ReadFile(path)
	assert.Error(t, err, "File doesn't exist, err.")

	content := []byte("pizza time")
	err = WriteFile(path, content)
	assert.NoError(t, err, "File write without fail")

	var b []byte
	b, err = ReadFile(path)
	assert.NoError(t, err, "File doesn't exist, err.")
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

	name, err := WriteTempFileToDir("", pattern, data, 0700)
	require.NoError(t, err)
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

func TestCopyFilesAndRename(t *testing.T) {
	var (
		src          = getTempDir(t, t.Name())
		sourceDir    = filepath.Join(src, "source-dir")
		sourceFile1  = filepath.Join(sourceDir, "file1")
		sourceFile2  = filepath.Join(sourceDir, "file2")
		destDir      = filepath.Join(src, "dest-dir")
		existingFile = filepath.Join(destDir, "file1")
		destFile2    = filepath.Join(destDir, "file2")
	)
	defer os.RemoveAll(src)

	err := Mkdir(sourceDir)
	require.NoError(t, err)
	err = Mkdir(destDir)
	require.NoError(t, err)

	err = os.WriteFile(sourceFile1, []byte("overwritten"), 0660)
	require.NoError(t, err)
	err = os.WriteFile(sourceFile2, []byte("new"), 0660)
	require.NoError(t, err)
	err = os.WriteFile(existingFile, []byte("original"), 0660)
	require.NoError(t, err)

	err = CopyAndRenameFiles(sourceDir, destDir)
	require.NoError(t, err)
	require.DirExists(t, destDir)
	assert.FileExists(t, destFile2)
	b, err := os.ReadFile(destFile2)
	require.NoError(t, err)
	assert.Equal(t, "new", string(b))
	assert.FileExists(t, existingFile)
	b, err = os.ReadFile(existingFile)
	require.NoError(t, err)
	assert.Equal(t, "overwritten", string(b))
}

func TestCopyFiles(t *testing.T) {
	var (
		src        = getTempDir(t, t.Name())
		dest       = getTempDir(t, strings.Join([]string{t.Name(), "destination"}, "-"))
		sourceDir  = filepath.Join(src, "test-dir")
		sourceFile = filepath.Join(src, "test-dir", "test-file")
		sourceLink = filepath.Join(src, "test-link")
		destDir    = filepath.Join(dest, "test-dir")
		destFile   = filepath.Join(filepath.Join(dest, "test-dir", "test-file"))
		destLink   = filepath.Join(dest, "test-link")
	)
	defer func() {
		os.RemoveAll(src)
		os.RemoveAll(dest)
	}()

	err := Mkdir(sourceDir)
	require.NoError(t, err)

	err = Touch(sourceFile)
	require.NoError(t, err)

	if runtime.GOOS != "windows" {
		// Symlink creation on Windows requires privledged create
		err := os.Symlink(sourceFile, sourceLink)
		require.NoError(t, err)
	}

	err = CopyFiles(src, dest)
	require.NoError(t, err)
	require.DirExists(t, dest)
	require.DirExists(t, destDir)
	require.FileExists(t, destFile)

	if runtime.GOOS != "windows" {
		require.FileExists(t, destLink)

		link, err := os.Readlink(destLink)
		require.NoError(t, err)
		require.Equal(t, sourceFile, link)
	}
}

type symlinkTestInfo struct {
	src,
	dest,
	srcDir,
	srcFile,
	srcLink,
	destFile,
	destLink string
}

func TestCopySymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip symlink test on Windows")
	}

	var (
		src  = getTempDir(t, t.Name())
		dest = getTempDir(t, strings.Join([]string{t.Name(), "destination"}, "-"))
	)

	info := symlinkTestInfo{
		src:      getTempDir(t, t.Name()),
		dest:     getTempDir(t, strings.Join([]string{t.Name(), "destination"}, "-")),
		srcDir:   filepath.Join(src, "bar"),
		srcFile:  filepath.Join(src, "bar", "foo"),
		srcLink:  filepath.Join(src, "foo"),
		destFile: filepath.Join(dest, "bar", "foo"),
		destLink: filepath.Join(dest, "foo"),
	}

	runSymlinkTest(t, info)
}

func TestCopySymlinkRelative(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skip symlink test on Windows")
	}

	dest := getTempDir(t, strings.Join([]string{t.Name(), "destination"}, "-"))
	info := symlinkTestInfo{
		src:      getTempDir(t, t.Name()),
		dest:     dest,
		srcDir:   "bar",
		srcFile:  "bar/foo",
		srcLink:  "foo",
		destFile: filepath.Join(dest, "bar", "foo"),
		destLink: filepath.Join(dest, "foo"),
	}

	runSymlinkTest(t, info)
}

func runSymlinkTest(t *testing.T, info symlinkTestInfo) {
	err := os.Chdir(info.src)
	require.NoError(t, err)

	err = Mkdir(info.srcDir)
	require.NoError(t, err)
	err = Touch(info.srcFile)
	require.NoError(t, err)

	content := "stuff"
	err = os.WriteFile(info.srcFile, []byte(content), 0644)
	require.NoError(t, err)

	err = os.Symlink(info.srcFile, info.srcLink)
	require.NoError(t, err)

	linkContent, err := os.ReadFile(info.srcLink)
	require.NoError(t, err)
	require.Equal(t, content, string(linkContent))

	err = CopyFile(info.srcFile, info.destFile)
	require.NoError(t, err)
	err = CopySymlink(info.srcLink, info.destLink)
	require.NoError(t, err)

	copiedLinkContent, err := os.ReadFile(info.destLink)
	require.NoError(t, err)
	require.Equal(t, content, string(copiedLinkContent))
}

func touchFile(t *testing.T, contents string, paths ...string) string {
	pd := filepath.Join(paths[:len(paths)-1]...)
	fp := filepath.Join(pd, paths[len(paths)-1])
	if pd != "" {
		err := MkdirUnlessExists(pd)
		require.NoError(t, err, "creating parent directory %s", pd)
	}
	err := os.WriteFile(fp, []byte(contents), 0666)
	require.NoError(t, err, "Touching %s", fp)
	return fp
}

func assertFileWithContent(t *testing.T, contents string, paths ...string) {
	fp := filepath.Join(paths...)
	res, err := os.ReadFile(fp)
	assert.NoError(t, err, "reading %s", fp)
	assert.Equal(t, contents, string(res))
}

func TestMoveAllFilesRecursively(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "activestatecli-test")
	require.NoError(t, err, "Created a temp dir")
	defer os.RemoveAll(tempDir)

	fromDir := filepath.Join(tempDir, "from")
	toDir := filepath.Join(tempDir, "to")

	var movedFiles []string
	var expected []string

	expected = append(expected, touchFile(t, "1", fromDir, "only_in_1", "t1"))
	expected = append(expected, touchFile(t, "1", fromDir, "in_1_and_2", "only_in_1"))
	expected = append(expected, touchFile(t, "1", fromDir, "in_1_and_2", "in_1_and_2"))
	expected = append(expected, touchFile(t, "1", fromDir, "root_in_1_only"))
	expected = append(expected, touchFile(t, "1", fromDir, "root_in_1_and_2"))
	expected = append(expected, filepath.Join(fromDir, "only_in_1"), filepath.Join(fromDir, "in_1_and_2"))
	touchFile(t, "2", toDir, "only_in_2", "t2")
	touchFile(t, "2", toDir, "in_1_and_2", "only_in_2")
	touchFile(t, "2", toDir, "in_1_and_2", "in_1_and_2")
	touchFile(t, "2", toDir, "root_in_2_only")
	touchFile(t, "2", toDir, "root_in_1_and_2")

	// Test that we handle symlinks to existing directories correctly
	if runtime.GOOS != "windows" {
		dirSymlink := filepath.Join(fromDir, "dirSymlink")
		err = os.Symlink(filepath.Join(".", "in_1_and_2"), dirSymlink)
		require.NoError(t, err)
		err = os.Symlink(filepath.Join(".", "in_1_and_2"), filepath.Join(toDir, "dirSymlink"))
		require.NoError(t, err)
		expected = append(expected, dirSymlink)
	}

	err = os.Chmod(filepath.Join(fromDir, "root_in_1_and_2"), 0440)
	require.NoError(t, err)
	err = os.Chmod(filepath.Join(toDir, "root_in_1_and_2"), 0440)
	require.NoError(t, err)

	err = MoveAllFilesRecursively(fromDir, toDir, func(from string, _ string) { movedFiles = append(movedFiles, from) })
	assert.NoError(t, err)

	assertFileWithContent(t, "1", toDir, "only_in_1", "t1")
	assertFileWithContent(t, "1", toDir, "in_1_and_2", "only_in_1")
	assertFileWithContent(t, "2", toDir, "only_in_2", "t2")
	assertFileWithContent(t, "2", toDir, "in_1_and_2", "only_in_2")
	assertFileWithContent(t, "1", toDir, "in_1_and_2", "in_1_and_2")
	assertFileWithContent(t, "2", toDir, "root_in_2_only")
	assertFileWithContent(t, "1", toDir, "root_in_1_and_2")

	assert.ElementsMatch(t, expected, movedFiles, "callback should have triggered for all files and directories")

	fp, err := os.Open(fromDir)
	require.NoError(t, err, "reading from dir")
	_, err = fp.Readdirnames(1)
	assert.Error(t, err, "reading dir contents %s", fromDir)
	assert.IsType(t, io.EOF, err)
}

func TestIsSameOrInsideOf(t *testing.T) {
	setSep := func(path string) string {
		return strings.ReplaceAll(path, "/", string(os.PathSeparator))
	}

	insideOf := resolvedPathContainsParent(setSep("../../internal/fileutils"), setSep("../../internal"))
	assert.True(t, insideOf)

	insideOf = resolvedPathContainsParent(setSep("../../internal/fileutils"), setSep("../../cmd"))
	assert.False(t, insideOf)

	insideOf = resolvedPathContainsParent(setSep("../../internalfileutils"), setSep("../../internal"))
	assert.False(t, insideOf)
}

func TestResolveUniquePath(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	require.NoError(t, err)

	defer os.RemoveAll(tempDir)

	targetPath := filepath.Join(tempDir, "target_long")

	err = Touch(targetPath)
	require.NoError(t, err)

	expectedPath := targetPath
	// On MacOS the os.MkdirTemp returns a symlink to the temporary directory
	if runtime.GOOS == "darwin" {
		expectedPath, err = filepath.EvalSymlinks(targetPath)
		require.NoError(t, err)
	} else if runtime.GOOS == "windows" {
		expectedPath, err = GetLongPathName(targetPath)
		require.NoError(t, err)
	}

	shortPath, err := GetShortPathName(targetPath)
	require.NoError(t, err, "Could not shorten path name.")

	sep := string(os.PathSeparator)

	cases := []struct {
		name string
		path string
	}{
		{"identity", targetPath},
		{"with slashes", tempDir + sep + "." + sep + "target_long" + sep},
		{"short path", shortPath},
	}

	if runtime.GOOS != "windows" {
		err = os.Symlink("target_long", filepath.Join(tempDir, "symlink"))
		require.NoError(t, err)
		cases = append(cases, struct {
			name string
			path string
		}{"symlink", filepath.Join(tempDir, "symlink")})
	}

	for _, c := range cases {
		t.Run(c.name, func(tt *testing.T) {
			res, err := ResolveUniquePath(c.path)
			assert.NoError(tt, err)
			assert.Equal(tt, expectedPath, res)
		})
	}

	t.Run("non-existent", func(tt *testing.T) {
		nonExistent := filepath.Join(tempDir, "non-existent")

		res, err := ResolveUniquePath(nonExistent)
		assert.NoError(tt, err)
		assert.Equal(tt, nonExistent, res)
	})
}

func TestPathsMatch(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("PathsMatch is only tested on macOS")
	}
	p1 := "/tmp"
	p2 := "/private/tmp"
	v, err := PathsMatch(p1, p2)
	require.NoError(t, err, errs.JoinMessage(err))

	v1, err := ResolvePath(p1)
	require.NoError(t, err, errs.JoinMessage(err))
	v2, err := ResolvePath(p2)
	require.NoError(t, err, errs.JoinMessage(err))

	require.True(t, v, "PathsMatch should return true, path1: %s, path2: %s", v1, v2)
}

func TestIsWritableFile(t *testing.T) {
	file, err := WriteTempFileToDir(
		"", t.Name(), []byte("Some data"), 0777,
	)
	if err != nil {
		t.Error(err)
	}

	if IsWritable(file) != true {
		t.Fatal("File should be writable")
	}

	err = os.Chmod(file, 0444)
	if err != nil {
		t.Error(err)
	}

	if IsWritable(file) != false {
		t.Fatal("File should no longer be writable")
	}
}

func TestIsWritableDir(t *testing.T) {
	pathWithPermission, err := user.HomeDir()
	if err != nil {
		t.Error(err)
	}
	if IsWritable(pathWithPermission) != true {
		t.Fatalf("Path should be writable: %s", pathWithPermission)
	}

	nonExistPathWithPermission := filepath.Join(pathWithPermission, funk.RandomString(10))
	if IsWritable(nonExistPathWithPermission) != true {
		t.Fatalf("Path should be writable: %s", nonExistPathWithPermission)
	}

	pathWithNoPermission := "/no-permission"
	if runtime.GOOS == "windows" {
		pathWithNoPermission = "C:\\Program Files\\No Permission"
	}
	if IsWritable(pathWithNoPermission) != false {
		t.Fatalf("Path should not be writable: %s", pathWithNoPermission)
	}
}

func TestCommonParentPath(t *testing.T) {
	tests := []struct {
		paths    []string
		expected string
	}{
		{
			paths:    []string{"./folder1/file.txt", "./folder1/subfolder/file.txt"},
			expected: "./folder1",
		},
		{
			paths:    []string{"./folder1/file.txt", "./folder2/file.txt"},
			expected: ".",
		},
		{
			paths:    []string{"./folder1/subfolder1/file.txt", "./folder1/subfolder2/file.txt"},
			expected: "./folder1",
		},
		{
			paths:    []string{"./folder1/", "./folder1/subfolder/"},
			expected: "./folder1",
		},
		{
			paths:    []string{"./folder1/file.txt", "./folder1/file.txt"},
			expected: "./folder1/file.txt",
		},
		{
			paths:    []string{"./folder1/file.txt", "./folder1/"},
			expected: "./folder1",
		},
		{
			paths:    []string{"./folder1/file.txt", "./folder2/"},
			expected: ".",
		},
		{
			paths:    []string{"./folder1/file.txt", "./folder1/subfolder/file.txt", "./folder1/subfolder2/file.txt"},
			expected: "./folder1",
		},
		{
			paths:    []string{"./folder1/file.txt", "./folder1/subfolder/file.txt", "./folder2/file.txt"},
			expected: ".",
		},
		{
			paths:    []string{"./folder1/file.txt", "./folder1/subfolder/file.txt", "./folder1/subfolder2/file.txt", "./folder1/subfolder3/file.txt"},
			expected: "./folder1",
		},
		{
			paths:    []string{"./folder1/file.txt", "./folder1/file.txt", "./folder1/file.txt"},
			expected: "./folder1/file.txt",
		},
		{
			paths:    []string{},
			expected: "",
		},
	}

	for _, test := range tests {
		t.Run(filepath.Join(test.paths...), func(t *testing.T) {
			result := CommonParentPath(test.paths)
			if result != test.expected {
				t.Errorf("CommonParentPath(%v) = %q; expected %q", test.paths, result, test.expected)
			}
		})
	}
}

func TestCommonParentPathx(t *testing.T) {
	tests := []struct {
		a, b     string
		expected string
	}{
		{"./folder1/file.txt", "./folder1/subfolder/file.txt", "./folder1"},
		{"./folder1/file.txt", "./folder2/file.txt", "."},
		{"./folder1/subfolder1/file.txt", "./folder1/subfolder2/file.txt", "./folder1"},
		{"./folder1/", "./folder1/subfolder/", "./folder1"},
		{"./folder1/file.txt", "./folder1/file.txt", "./folder1/file.txt"},
		{"./folder1/file.txt", "./folder1/", "./folder1"},
		{"./folder1/file.txt", "./folder2/", "."},
		{"", "./folder1/file.txt", ""},
		{"./folder1/file.txt", "", ""},
		{"", "", ""},
	}

	for x, test := range tests {
		result := commonParentPath(test.a, test.b)
		if result != test.expected {
			t.Errorf("%d: commonParentPath(%q, %q) = %q; expected %q", x, test.a, test.b, result, test.expected)
		}
	}
}
