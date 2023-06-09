package localcommit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalCommit(t *testing.T) {
	tempDir := fileutils.TempDirUnsafe()
	defer os.RemoveAll(tempDir)

	_, err := Get(tempDir)
	assert.Error(t, err)
	assert.True(t, IsFileDoesNotExistError(err), "error is not a FileDoesNotExist error")

	commitID := "00010001-0001-0001-0001-000100010001"
	err = Set(tempDir, commitID)
	require.NoError(t, err)

	commitIdFile := filepath.Join(tempDir, constants.ProjectConfigDirName, constants.CommitIdFileName)
	require.FileExists(t, commitIdFile)
	assert.Equal(t, commitID, string(fileutils.ReadFileUnsafe(commitIdFile)))
	localCommitID, err := Get(tempDir)
	require.NoError(t, err)
	assert.Equal(t, commitID, localCommitID.String())

	// Test creating new .gitignore.
	gitIgnoreFile := filepath.Join(tempDir, ".gitignore")
	assert.NoFileExists(t, gitIgnoreFile)
	err = AddToGitIgnore(tempDir)
	require.NoError(t, err)
	require.FileExists(t, gitIgnoreFile)
	assert.Contains(t, string(fileutils.ReadFileUnsafe(gitIgnoreFile)), fmt.Sprintf("%s/%s", constants.ProjectConfigDirName, constants.CommitIdFileName))

	// Test append to existing .gitignore.
	err = os.Remove(gitIgnoreFile)
	require.NoError(t, err)
	err = fileutils.WriteFile(gitIgnoreFile, []byte("foo\nbar\nbaz"))
	require.NoError(t, err)
	err = AddToGitIgnore(tempDir)
	require.NoError(t, err)
	assert.Contains(t, string(fileutils.ReadFileUnsafe(gitIgnoreFile)), "foo\nbar\nbaz")
	assert.Contains(t, string(fileutils.ReadFileUnsafe(gitIgnoreFile)), fmt.Sprintf("\n%s/%s", constants.ProjectConfigDirName, constants.CommitIdFileName))

	// Test multiple calls to append to .gitignore do not add multiple files.
	err = AddToGitIgnore(tempDir)
	require.NoError(t, err)
	contents := string(fileutils.ReadFileUnsafe(gitIgnoreFile))
	added := locale.Tr("commit_id_gitignore", constants.ProjectConfigDirName, constants.CommitIdFileName)
	assert.Equal(t, 1, strings.Count(contents, added), "multiple commit ID files added to .gitignore")
}
