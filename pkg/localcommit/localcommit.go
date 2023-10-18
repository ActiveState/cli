package localcommit

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/go-openapi/strfmt"
)

type ErrLocalCommitFile struct {
	*locale.LocalizedError // for backwards compatibility with runners that don't implement rationalizers
	errorMsg               string
	IsDoesNotExist         bool
	File                   string
}

type projecter interface {
	Dir() string
	LegacyCommitID() string
}

func (e *ErrLocalCommitFile) Error() string {
	return e.errorMsg
}

func IsFileDoesNotExistError(err error) bool {
	var errLocalCommit *ErrLocalCommitFile
	if errors.As(err, &errLocalCommit) {
		return errLocalCommit.IsDoesNotExist
	}
	return false
}

func getCommitFile(projectDir string) string {
	return filepath.Join(projectDir, constants.ProjectConfigDirName, constants.CommitIdFileName)
}

func Get(projectDir string) (strfmt.UUID, error) {
	configDir := filepath.Join(projectDir, constants.ProjectConfigDirName)
	commitFile := getCommitFile(projectDir)
	if !fileutils.DirExists(configDir) || !fileutils.TargetExists(commitFile) {
		return "", &ErrLocalCommitFile{
			locale.NewError("err_local_commit_file", commitFile),
			"local commit file does not exist",
			true, commitFile}
	}

	b, err := fileutils.ReadFile(commitFile)
	if err != nil {
		return "", &ErrLocalCommitFile{
			locale.NewError("err_local_commit_file", commitFile),
			"local commit could not be read",
			false, commitFile}
	}

	commitID := string(b)
	if !strfmt.IsUUID(commitID) {
		return "", &ErrLocalCommitFile{
			locale.NewError("err_local_commit_file", commitFile),
			"local commit is not uuid formatted",
			false, commitFile}
	}

	return strfmt.UUID(commitID), nil
}

// GetCompatible returns the given project's commit ID in either the new format (commit file), or
// the old format (activestate.yaml).
func GetCompatible(proj projecter) (strfmt.UUID, error) {
	if commitID, err := Get(proj.Dir()); err == nil {
		return commitID, nil
	} else if IsFileDoesNotExistError(err) {
		return strfmt.UUID(proj.LegacyCommitID()), nil
	} else {
		return "", err
	}
}

func Set(projectDir, commitID string) error {
	if !strfmt.IsUUID(commitID) {
		return locale.NewInputError("err_commit_id_invalid", commitID)
	}

	commitFile := getCommitFile(projectDir)
	err := fileutils.WriteFile(commitFile, []byte(commitID))
	if err != nil {
		return locale.WrapError(err, "err_set_commit_id", "Unable to set your project runtime's commit ID")
	}
	return nil
}

func AddToGitIgnore(projectDir string) error {
	gitIgnore := filepath.Join(projectDir, ".gitignore")
	if !fileutils.TargetExists(gitIgnore) {
		err := fileutils.WriteFile(gitIgnore, []byte(locale.Tr("commit_id_gitignore", constants.ProjectConfigDirName, constants.CommitIdFileName)))
		if err != nil {
			return locale.WrapError(err, "err_commit_id_gitignore_create",
				"Unable to create a .gitignore file with your project runtime's commit ID file in it")
		}
		return nil
	}

	b, err := fileutils.ReadFile(gitIgnore)
	if err != nil {
		return locale.WrapError(err, "err_commit_id_gitignore_read", "Unable to read .gitignore file")
	}
	if bytes.Contains(b, []byte(fmt.Sprintf("%s/%s", constants.ProjectConfigDirName, constants.CommitIdFileName))) {
		return nil // already done
	}
	newline := "\n"
	if crlf := bytes.IndexByte(b, '\r'); crlf != -1 {
		newline = "\r" + newline
	}
	b = append(b, []byte(newline)...)
	b = append(b, []byte(locale.Tr("commit_id_gitignore", constants.ProjectConfigDirName, constants.CommitIdFileName))...)

	err = fileutils.WriteFile(gitIgnore, b)
	if err != nil {
		return locale.WrapError(err, "err_commit_id_gitignore_add",
			"Unable to add your project runtime's commit ID file to .gitignore")
	}

	return nil
}
