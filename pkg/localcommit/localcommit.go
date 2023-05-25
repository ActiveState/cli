package localcommit

import (
	"bytes"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/go-openapi/strfmt"
)

type FileDoesNotExistError struct{ *locale.LocalizedError }

func IsFileDoesNotExistError(err error) bool {
	return errs.Matches(err, &FileDoesNotExistError{})
}

func Get(projectDir string) (string, error) {
	configDir := filepath.Join(projectDir, constants.ProjectConfigDirName)
	commitFile := filepath.Join(configDir, constants.CommitIdFileName)
	if !fileutils.DirExists(configDir) || !fileutils.FileExists(commitFile) {
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

func Set(projectDir, commitID string) error {
	if !strfmt.IsUUID(commitID) {
		return locale.NewError("err_commit_id_invalid", commitID)
	}

	updateGitIgnore := shouldAddToGitIgnore(projectDir)
	commitFile := filepath.Join(projectDir, constants.ProjectConfigDirName, constants.CommitIdFileName)
	err := fileutils.WriteFile(commitFile, []byte(commitID))
	if err != nil {
		return locale.WrapError(err, "err_set_commit_id", "Unable to set your project runtime's commit ID")
	}
	if updateGitIgnore {
		addToGitIgnore(projectDir)
	}
	return nil
}

func shouldAddToGitIgnore(projectDir string) bool {
	files, err := fileutils.ListDir(projectDir, true)
	if err != nil {
		multilog.Error("Cannot determine whether to add runtime commit ID file to .gitignore: %v", err)
		return false
	}

	if len(files) == 0 {
		return true // fresh checkout
	}

	for _, file := range files {
		if file.Name() == ".git" {
			return true // project is under Git revision control
		}
	}

	return false
}

func addToGitIgnore(projectDir string) error {
	gitIgnore := filepath.Join(projectDir, ".gitignore")
	if !fileutils.TargetExists(gitIgnore) {
		err := fileutils.WriteFile(gitIgnore, []byte(locale.T("commit_id_gitignore")))
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
	newline := "\n"
	if crlf := bytes.IndexByte(b, '\r'); crlf != -1 {
		newline = "\r" + newline
	}
	b = append(b, []byte(newline)...)
	b = append(b, []byte(locale.T("commit_id_gitignore"))...)

	err = fileutils.WriteFile(gitIgnore, b)
	if err != nil {
		return locale.WrapError(err, "err_commit_id_gitignore_add",
			"Unable to add your project runtime's commit ID file to .gitignore")
	}

	return nil
}
