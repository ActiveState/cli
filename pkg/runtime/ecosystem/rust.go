package ecosystem

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/osutils"

	"github.com/ActiveState/cli/pkg/buildplan"
)

type Rust struct {
	runtimeDir     string
	crateCacheName string // e.g. index.crates.io-6f16d22bbaa15001f
}

func (e *Rust) Init(runtimePath string, buildplan *buildplan.BuildPlan) error {
	e.runtimeDir = runtimePath

	var err error
	e.crateCacheName, err = getCacheDirName(runtimePath)
	switch {
	case err != nil:
		return errs.Wrap(err, "Unable to get cargo registry cache dir")
	case e.crateCacheName == "":
		return locale.NewError("rust_cache_not_found", "The Rust runtime does not have a cache directory for the crates.io index")
	}

	return nil
}

func (e *Rust) Namespaces() []string {
	return []string{"language/rust"}
}

// Add copies the "download" artifact to the Cargo cache directory, renaming it to
// "<name>-<version>.crate" format.
// We also inject the CARGO_HOME environment variable into runtime.json so cargo will look in the
// right place for our artifacts instead of downloading its own.
func (e *Rust) Add(artifact *buildplan.Artifact, artifactSrcPath string) ([]string, error) {
	installedFiles := []string{}

	files, err := fileutils.ListDir(artifactSrcPath, false)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to read artifact source directory")
	}

	for _, file := range files {
		if file.Name() == "runtime.json" {
			err = injectEnvVar(file.AbsolutePath(), "CARGO_HOME", "${INSTALLDIR}/usr/cargo")
			if err != nil {
				return nil, errs.Wrap(err, "Unable to add CARGO_HOME to runtime.json")
			}
			continue
		}
		if file.Name() != "download" {
			continue
		}

		name := fmt.Sprintf("%s-%s.crate", artifact.Name(), artifact.Version())
		relativeCrateCacheDir := filepath.Join("usr", "cargo", "registry", "cache", e.crateCacheName)
		relativeInstalledFile := filepath.Join(relativeCrateCacheDir, name)
		installedFile := filepath.Join(e.runtimeDir, relativeInstalledFile)
		err = fileutils.CopyFile(file.AbsolutePath(), installedFile)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to copy artifact crate into cache directory")
		}
		installedFiles = append(installedFiles, relativeInstalledFile)
	}

	return installedFiles, nil
}

func (e *Rust) Remove(artifact *buildplan.Artifact) error {
	return nil // TODO: CP-956
}

func (e *Rust) Apply() error {
	return nil
}

// getCacheDirName returns Cargo's crates.io registry cache directory for this runtime.
// Note: at the time of writing, the runtime has a usr/registry installed by the langauge core that
// does not appear to be used, nor can it since Cargo's registry index directories have location-
// dependent checksum suffixes, and the runtime's suffix was created at build-time.
func getCacheDirName(runtimePath string) (string, error) {
	cachePrefix := filepath.Join(runtimePath, "usr", "cargo", "registry", "cache")
	if !fileutils.DirExists(cachePrefix) {
		// Invoke cargo with a bogus "search" command that will fetch the crates.io registry index.
		// We can use the registry index location to infer where the cache directory should be.
		_, stderr, err := osutils.ExecSimple("cargo", []string{"search"}, []string{"CARGO_HOME=" + filepath.Join(runtimePath, "usr", "cargo")})
		if err != nil {
			return "", errs.Wrap(err, "Error running cargo: %s", stderr)
		}
		// Cargo's registry index directories have location-dependent checksum suffixes. Read it.
		entries, err := fileutils.ListDirSimple(filepath.Join(runtimePath, "usr", "cargo", "registry", "index"), true)
		if err != nil {
			return "", errs.Wrap(err, "Unable to list contents of runtime usr/cargo/registry/index")
		}
		for _, entry := range entries {
			if strings.Contains(entry, "index.crates.io-") {
				name := filepath.Base(entry)
				err = fileutils.Mkdir(filepath.Join(cachePrefix, name))
				if err != nil {
					return "", errs.Wrap(err, "Unable to make cache prefix")
				}
				return name, nil
			}
		}
	}

	// We already did the song and dance to create the location-specific cache directory, so just
	// read it from the filesystem.
	entries, err := fileutils.ListDirSimple(cachePrefix, true)
	if err != nil {
		return "", errs.Wrap(err, "Unable to list contents of runtime usr/cargo/registry/cache")
	}
	for _, entry := range entries {
		if strings.Contains(entry, "index.crates.io-") {
			return filepath.Base(entry), nil
		}
	}

	return "", nil
}
