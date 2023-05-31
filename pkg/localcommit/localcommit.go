package localcommit

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/go-openapi/strfmt"
)

type FileDoesNotExistError struct{ *locale.LocalizedError }

func IsFileDoesNotExistError(err error) bool {
	return errs.Matches(err, &FileDoesNotExistError{})
}

func getCommitFile(projectDir string) string {
	return filepath.Join(projectDir, constants.ProjectConfigDirName, constants.CommitIdFileName)
}

func Get(projectDir string) (string, error) {
	configDir := filepath.Join(projectDir, constants.ProjectConfigDirName)
	commitFile := getCommitFile(projectDir)
	if !fileutils.DirExists(configDir) || !fileutils.TargetExists(commitFile) {
		return "", &FileDoesNotExistError{locale.NewError("err_commit_file_does_not_exist",
			"Your project runtime's commit ID file '{{.V0}}' does not exist", commitFile)}
	}

	b, err := fileutils.ReadFile(commitFile)
	if err != nil {
		return "", locale.WrapError(err, "err_get_commit_file", "Could not read your project runtime's commit ID file")
	}

	commitID := string(b)
	if !strfmt.IsUUID(commitID) {
		return "", locale.NewError("err_commit_id_invalid", commitID)
	}

	return commitID, nil
}

func GetUUID(projectDir string) (strfmt.UUID, error) {
	commitID, err := Get(projectDir)
	if err != nil {
		return "", errs.Wrap(err, "Unable to get local commit")
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
