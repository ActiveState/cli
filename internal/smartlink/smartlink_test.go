package smartlink

import (
	"os"
	"path/filepath"
	"runtime"
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

	err = LinkContents(srcDir, destDir, nil)
	if runtime.GOOS == "windows" {
		require.Error(t, err)
		return // hard links between directories is not allowed on Windows
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
