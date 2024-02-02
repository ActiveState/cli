package localcommit

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/go-openapi/strfmt"
)

type ErrLocalCommitFile struct {
	*locale.LocalizedError // for backwards compatibility with runners that don't implement rationalizers
	errorMsg               string
	IsDoesNotExist         bool
	File                   string
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

type migrationHandler interface {
	Migrate(projectDir string) (strfmt.UUID, error)
	Set(projectDir, commitID string) error
}

var handler migrationHandler = nil

func RegisterMigrator(h migrationHandler) {
	handler = h
}

func Get(projectDir string) (strfmt.UUID, error) {
	configDir := filepath.Join(projectDir, constants.ProjectConfigDirName)
	commitFile := getCommitFile(projectDir)
	if !fileutils.DirExists(configDir) || !fileutils.TargetExists(commitFile) {
		// If we have a legacy handler we use the commitID from there and store it to the localcommit file for future use.
		if handler != nil {
			commitId, err := handler.Migrate(projectDir)
			if err != nil {
				return "", errs.Wrap(err, "Could not retrieve local commit ID through handler")
			}
			if err := AddToGitIgnore(projectDir); err != nil {
				if !errors.Is(err, fs.ErrPermission) {
					multilog.Error("Unable to add local commit file to .gitignore: %v", err)
				}
			}
			if err := Set(projectDir, commitId.String()); err != nil {
				return "", errs.Wrap(err, "Storing commit after migration failed")
			}
			return commitId, nil
		}

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

func Set(projectDir, commitID string) error {
	if !strfmt.IsUUID(commitID) {
		return locale.NewInputError("err_commit_id_invalid", commitID)
	}

	commitFile := getCommitFile(projectDir)
	err := fileutils.WriteFile(commitFile, []byte(commitID))
	if err != nil {
		return locale.WrapError(err, "err_set_commit_id", "Unable to set your project runtime's commit ID")
	}

	// If we have a legacy handler we also want to send the commit there.
	// But since the legacy handler is not the source of truth we don't want to break on errors.
	if handler != nil {
		if err := handler.Set(projectDir, commitID); err != nil {
			multilog.Error("Could not set legacy commit ID through handler: %s", errs.JoinMessage(err))
		}
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
