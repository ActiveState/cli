package smartlink

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/stretchr/testify/require"
)

func TestLinkContentsWithCircularLink(t *testing.T) {
	srcDir, err := os.MkdirTemp("", "src")
	require.NoError(t, err)
	defer os.RemoveAll(srcDir)

	destDir, err := os.MkdirTemp("", "dest")
	require.NoError(t, err)
	defer os.RemoveAll(destDir)

	// Create test file structure:
	// src/
	//   ├── regular.txt
	//   └── subdir/
	//        ├── circle -> subdir (circular link)
	//        └── subfile.txt

	testFile := filepath.Join(srcDir, "regular.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	subDir := filepath.Join(srcDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	subFile := filepath.Join(subDir, "subfile.txt")
	err = os.WriteFile(subFile, []byte("sub content"), 0644)
	require.NoError(t, err)

	circularLink := filepath.Join(subDir, "circle")
	err = os.Symlink(subDir, circularLink)
	require.NoError(t, err)

	err = LinkContents(srcDir, destDir)
	if runtime.GOOS == "windows" {
		require.Error(t, err)
		return // hard links to directories are not allowed on Windows
	}
	require.NoError(t, err)

	// Verify file structure.
	destFile := filepath.Join(destDir, "regular.txt")
	require.FileExists(t, destFile)
	content, err := os.ReadFile(destFile)
	require.NoError(t, err)
	require.Equal(t, "test content", string(content))

	destSubFile := filepath.Join(destDir, "subdir", "subfile.txt")
	require.FileExists(t, destSubFile)
	subContent, err := os.ReadFile(destSubFile)
	require.NoError(t, err)
	require.Equal(t, "sub content", string(subContent))

	destCircular := filepath.Join(destDir, "subdir", "circle")
	require.FileExists(t, destCircular)
	target, err := fileutils.ResolveUniquePath(destCircular)
	require.NoError(t, err)
	srcCircular := filepath.Join(srcDir, "subdir")
	if runtime.GOOS == "darwin" {
		srcCircular, err = fileutils.ResolveUniquePath(srcCircular) // needed for full $TMPDIR resolution
		require.NoError(t, err)
	}
	require.Equal(t, target, srcCircular)
}

func TestLinkSkipsExistingTarget(t *testing.T) {
	srcDir, destDir := t.TempDir(), t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("new"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(destDir, "file.txt"), []byte("existing"), 0644))

	require.NoError(t, Link(filepath.Join(srcDir, "file.txt"), filepath.Join(destDir, "file.txt")))

	content, err := os.ReadFile(filepath.Join(destDir, "file.txt"))
	require.NoError(t, err)
	require.Equal(t, "existing", string(content), "an existing target must be left intact")
}

// TestLinkContentsConcurrentSameDest links several sources that provide the same files into one
// destination at once, as concurrent artifact deployments do. Each file must be linked exactly once
// (first one wins) without any of the deployments failing.
func TestLinkContentsConcurrentSameDest(t *testing.T) {
	const sources, files = 8, 50
	destDir := t.TempDir()
	srcDirs := make([]string, sources)
	for i := range srcDirs {
		srcDirs[i] = t.TempDir()
		for f := 0; f < files; f++ {
			path := filepath.Join(srcDirs[i], "sub", fmt.Sprintf("f%02d.txt", f))
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
			require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf("src%d", i)), 0644))
		}
	}

	var wg sync.WaitGroup
	errs := make([]error, sources)
	for i, src := range srcDirs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = LinkContents(src, destDir)
		}()
	}
	wg.Wait()
	for i, err := range errs {
		require.NoError(t, err, "deployment %d failed", i)
	}

	for f := 0; f < files; f++ {
		content, err := os.ReadFile(filepath.Join(destDir, "sub", fmt.Sprintf("f%02d.txt", f)))
		require.NoError(t, err)
		require.Regexp(t, `^src\d$`, string(content))
	}
	for i, src := range srcDirs {
		content, err := os.ReadFile(filepath.Join(src, "sub", "f00.txt"))
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf("src%d", i), string(content), "a source file was modified")
	}
}

// TestLinkContentsEquivalence pins down how each kind of entry is linked: regular files and the
// contents of regular directories link to the file in src, a symlink to a file links to the file it
// points at, and a symlink to a directory is linked as the symlink itself (see Link).
func TestLinkContentsEquivalence(t *testing.T) {
	srcDir, destDir := t.TempDir(), t.TempDir()
	srcDir, err := fileutils.ResolveUniquePath(srcDir) // resolve $TMPDIR symlinks on macOS
	require.NoError(t, err)

	// src/
	//   ├── top.txt
	//   ├── dir/nested/deep.txt
	//   ├── file-link -> dir/nested/deep.txt
	//   └── dir-link -> dir/nested
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "top.txt"), []byte("top"), 0644))
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "dir", "nested"), 0755))
	deep := filepath.Join(srcDir, "dir", "nested", "deep.txt")
	require.NoError(t, os.WriteFile(deep, []byte("deep"), 0644))
	withSymlinks := runtime.GOOS != "windows" // creating symlinks needs privileges on Windows
	if withSymlinks {
		require.NoError(t, os.Symlink(deep, filepath.Join(srcDir, "file-link")))
		require.NoError(t, os.Symlink(filepath.Join(srcDir, "dir", "nested"), filepath.Join(srcDir, "dir-link")))
	}

	require.NoError(t, LinkContents(srcDir, destDir))

	sameFile := func(a, b string) {
		t.Helper()
		ai, err := os.Stat(a)
		require.NoError(t, err)
		bi, err := os.Stat(b)
		require.NoError(t, err)
		require.True(t, os.SameFile(ai, bi), "%s should be linked to %s", a, b)
	}
	sameFile(filepath.Join(destDir, "top.txt"), filepath.Join(srcDir, "top.txt"))
	sameFile(filepath.Join(destDir, "dir", "nested", "deep.txt"), deep)
	require.True(t, fileutils.IsDir(filepath.Join(destDir, "dir", "nested")))
	require.False(t, fileutils.IsSymlink(filepath.Join(destDir, "dir")), "regular directories are recreated, not linked")

	if withSymlinks {
		// A symlink to a file links to the file it points at.
		target, err := os.Readlink(filepath.Join(destDir, "file-link"))
		require.NoError(t, err)
		require.Equal(t, deep, target)

		// A symlink to a directory is linked as the symlink itself, not followed.
		target, err = os.Readlink(filepath.Join(destDir, "dir-link"))
		require.NoError(t, err)
		require.Equal(t, filepath.Join(srcDir, "dir-link"), target)
	}
}
