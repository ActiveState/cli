package e2e

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

// Dirs represents directories that are temporarily created for this end-to-end testing session
type Dirs struct {
	base string
	// Config is where configuration files are stored
	Config string
	// Cache is the directory where cached files including downloaded artifacts are stored
	Cache string
	// Bin is the directory where executables are stored
	Bin string
	// Work is the working directory where the activestate.yaml file would live, and that is the PWD for tested console processes
	Work string
}

// NewDirs creates all temprorary directories
func NewDirs(base string) (*Dirs, error) {
	if base == "" {
		tmpDir, err := ioutil.TempDir("", "")
		if err != nil {
			return nil, err
		}
		base = tmpDir
	}

	config := filepath.Join(base, "config")
	cache := filepath.Join(base, "cache")
	bin := filepath.Join(base, "bin")
	work := filepath.Join(base, "work")

	subdirs := []string{config, cache, bin, work}
	for _, subdir := range subdirs {
		if err := os.MkdirAll(subdir, 0700); err != nil {
			return nil, err
		}
	}

	dirs := Dirs{
		base:   base,
		Config: config,
		Cache:  cache,
		Bin:    bin,
		Work:   work,
	}

	return &dirs, nil
}

// Close removes the temporary directories
func (d *Dirs) Close() error {
	return os.RemoveAll(d.base)
}
