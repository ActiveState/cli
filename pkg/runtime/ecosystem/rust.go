package ecosystem

import (
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/unarchiver"

	"github.com/ActiveState/cli/pkg/buildplan"
)

type Rust struct {
	runtimeDir string
	vendorDir  string
}

func (e *Rust) Init(runtimePath string, buildplan *buildplan.BuildPlan) error {
	e.runtimeDir = runtimePath
	e.vendorDir = filepath.Join("usr", "cargo", "vendor")
	err := fileutils.MkdirUnlessExists(filepath.Join(e.runtimeDir, e.vendorDir))
	if err != nil {
		return errs.Wrap(err, "Unable to create cargo vendor directory")
	}
	return nil
}

func (e *Rust) Namespaces() []string {
	return []string{"language/rust"}
}

// Unpack the crate into the vendor directory.
// We also inject the CARGO_HOME environment variable into runtime.json so cargo will look for our
// vendored artifacts instead of downloading its own.
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

		relativeVendored := filepath.Join(e.vendorDir, artifact.Name())
		absVendored := filepath.Join(e.runtimeDir, relativeVendored)

		// Delete any previously vendored crate.
		if fileutils.DirExists(absVendored) {
			err = os.RemoveAll(absVendored)
			if err != nil {
				return nil, errs.Wrap(err, "Unable to remove previously unpacked crate")
			}
		}

		// Unpacked crates contain a single <name>-<version> folder.
		// That folder needs to be renamed to just <name> for Cargo to recognize it.
		ua := unarchiver.NewTarGz()
		unpackDir := filepath.Join(e.runtimeDir, e.vendorDir)
		f, err := ua.PrepareUnpacking(file.AbsolutePath(), unpackDir)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to prepare for unpacking downloaded crate")
		}
		err = ua.Unarchive(f, unpackDir)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to unpack downloaded crate")
		}
		err = os.Rename(filepath.Join(unpackDir, artifact.Name()+"-"+artifact.Version()), absVendored)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to rename unpacked crate")
		}
		// Cargo needs a checksum file too. We cannot produce a real one, but it accepts an empty one.
		err = fileutils.WriteFile(filepath.Join(absVendored, ".cargo-checksum.json"), []byte(`{"files":{}}`))
		if err != nil {
			return nil, errs.Wrap(err, "Unable to write empty checksum")
		}

		installedFiles = append(installedFiles, relativeVendored)
	}

	return installedFiles, nil
}

func (e *Rust) Remove(name, version string, installedFiles []string) (rerr error) {
	for _, dir := range installedFiles {
		if !fileutils.DirExists(dir) {
			continue
		}
		err := os.RemoveAll(dir)
		if err != nil {
			rerr = errs.Pack(rerr, errs.Wrap(err, "Unable to remove directory for '%s': %s", name, dir))
		}
	}
	return rerr
}

// configFileContents replaces the crates.io source with our vendored crates.
// Note: the directory is relative to "e.runtimeDir/usr".
const configFileContents = `
[source.crates-io]
replace-with = "vendor"

[source.vendor]
directory = "cargo/vendor"
`

// Create a Cargo config file that tells Cargo to use our vendored crates instead of downloading
// its own.
func (e *Rust) Apply() error {
	configFile := filepath.Join(e.runtimeDir, "usr", "cargo", "config.toml")
	if !fileutils.TargetExists(configFile) {
		err := fileutils.WriteFile(configFile, []byte(configFileContents))
		if err != nil {
			return errs.Wrap(err, "Unable to write cargo config file")
		}
	}
	return nil
}
