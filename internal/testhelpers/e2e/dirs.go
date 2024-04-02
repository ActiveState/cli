package e2e

import (
	"os"
	"path/filepath"
)

// Dirs represents directories that are temporarily created for this end-to-end testing session
type Dirs struct {
	Base string
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
	// SockRoot is the directory for the state service's socket file
	SockRoot string
	// HomeDir is used as the test user's home directory
	HomeDir string
	// TempDir is the directory where temporary files are stored
	TempDir string
}

// NewDirs creates all temporary directories
func NewDirs(base string) (*Dirs, error) {
	if base == "" {
		tmpDir, err := os.MkdirTemp("", "")
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
	sockRoot := filepath.Join(base, "sock")
	homeDir := filepath.Join(base, "home")
	tempDir := filepath.Join(base, "temp")

	subdirs := []string{config, cache, bin, work, defaultBin, sockRoot, homeDir, tempDir}
	for _, subdir := range subdirs {
		if err := os.MkdirAll(subdir, 0700); err != nil {
			return nil, err
		}
	}

	dirs := Dirs{
		Base:       base,
		Config:     config,
		Cache:      cache,
		Bin:        bin,
		Work:       work,
		DefaultBin: defaultBin,
		SockRoot:   sockRoot,
		HomeDir:    homeDir,
		TempDir:    tempDir,
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
