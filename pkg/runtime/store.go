package runtime

import (
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
)

func (r *Runtime) loadHash() error {
	path := filepath.Join(r.path, configDir, hashFile)
	if !fileutils.TargetExists(path) {
		return nil
	}

	hash, err := fileutils.ReadFile(path)
	if err != nil {
		return errs.Wrap(err, "Failed to read hash file")
	}

	r.hash = string(hash)
	return nil
}

func (r *Runtime) saveHash(hash string) error {
	path := filepath.Join(r.path, configDir, hashFile)
	if !fileutils.TargetExists(path) {
		return errs.New("Hash file does not exist")
	}

	if err := fileutils.WriteFile(path, []byte(hash)); err != nil {
		return errs.Wrap(err, "Failed to write hash file")
	}

	return nil
}
