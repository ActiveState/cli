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
	// DefaultBin is the bin directory for our default installation
	DefaultBin string
	// InstallerBin is the direcotry where our installer executable is kept
	InstallerBin string
}

// NewDirs creates all temporary directories
func NewDirs(base string) (*Dirs, error) {
	if base == "" {
		tmpDir, err := ioutil.TempDir("", "")
		if err != nil {
			return nil, err
		}
		base = tmpDir
	}

	cache := filepath.Join(base, "cache")
	config := filepath.Join(base, "config")
	bin := filepath.Join(base, "bin")
	work := filepath.Join(base, "work")
	defaultBin := filepath.Join(base, "cache", "bin")
	installerBin := filepath.Join(base, "installer")

	subdirs := []string{config, cache, bin, work, defaultBin, installerBin}
	for _, subdir := range subdirs {
		if err := os.MkdirAll(subdir, 0700); err != nil {
			return nil, err
		}
	}

	dirs := Dirs{
		base:         base,
		Config:       config,
		Cache:        cache,
		Bin:          bin,
		Work:         work,
		DefaultBin:   defaultBin,
		InstallerBin: installerBin,
	}

	return &dirs, nil
}

// Close removes the temporary directories
func (d *Dirs) Close() error {
	subdirs := []string{d.Bin, d.Config, d.Work, d.Cache}
	for _, subdir := range subdirs {
		if err := os.RemoveAll(subdir); err != nil {
			return err
		}
	}
	return nil
}
