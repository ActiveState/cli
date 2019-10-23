// +build !windows

package fileutils

import (
	"os"
	"syscall"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
)

// IsExecutable determines if the file at the given path has any execute permissions.
// This function does not care whether the current user can has enough privilege to
// execute the file.
func IsExecutable(path string) bool {
	stat, err := os.Stat(path)
	return err == nil && (stat.Mode()&(0111) > 0)
}

func copyPermissions(fileInfo, entry os.FileInfo, dest string) *failures.Failure {
	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return failures.FailOS.New(locale.T("TODO:"))
	}

	if err := os.Lchown(dest, int(stat.Uid), int(stat.Gid)); err != nil {
		return failures.FailOS.Wrap(err)
	}

	return nil
}
