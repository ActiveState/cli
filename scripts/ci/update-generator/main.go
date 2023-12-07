package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mholt/archiver"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/internal/updater"
)

var (
	rootPath         = environment.GetRootPathUnsafe()
	defaultBuildDir  = filepath.Join(rootPath, "build")
	defaultInputDir  = filepath.Join(defaultBuildDir, "payload", constants.ToplevelInstallArchiveDir)
	defaultOutputDir = filepath.Join(rootPath, "public")
)

func main() {
	if !condition.InUnitTest() {
		err := run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s error: %v", os.Args[0], errs.JoinMessage(err))
			os.Exit(1)
		}
	}
}

func fetchPlatform() string {
	goos := os.Getenv("GOOS")
	goarch := os.Getenv("GOARCH")
	if goos != "" && goarch != "" {
		return goos + "-" + goarch
	}
	return runtime.GOOS + "-" + runtime.GOARCH
}

func generateSha256(path string) string {
	hasher := sha256.New()
	b, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	hasher.Write(b)
	return hex.EncodeToString(hasher.Sum(nil))
}

func archiveMeta() (archiveMethod archiver.Archiver, ext string) {
	if runtime.GOOS == "windows" {
		return archiver.NewZip(), ".zip"
	}
	return archiver.NewTarGz(), ".tar.gz"
}

func createUpdate(outputPath, channel, version, versionNumber, platform, target string) error {
	relChannelPath := filepath.Join(channel, platform)
	relVersionedPath := filepath.Join(channel, versionNumber, platform)
	_ = os.MkdirAll(filepath.Join(outputPath, relChannelPath), 0o755)
	_ = os.MkdirAll(filepath.Join(outputPath, relVersionedPath), 0o755)

	archive, archiveExt := archiveMeta()
	relArchivePath := filepath.Join(relVersionedPath, fmt.Sprintf("state-%s-%s%s", platform, version, archiveExt))
	archivePath := filepath.Join(outputPath, relArchivePath)

	// Remove archive path if it already exists
	_ = os.Remove(archivePath)
	// Create main archive
	fmt.Printf("Creating %s\n", archivePath)
	if err := archive.Archive([]string{target}, archivePath); err != nil {
		return errs.Wrap(err, "Archiving failed")
	}

	avUpdate := updater.NewAvailableUpdate(channel, version, platform, filepath.ToSlash(relArchivePath), generateSha256(archivePath), "")
	b, err := json.MarshalIndent(avUpdate, "", "    ")
	if err != nil {
		return errs.Wrap(err, "Failed to marshal AvailableUpdate information.")
	}

	infoPath := filepath.Join(outputPath, relChannelPath, "info.json")
	fmt.Printf("Creating %s\n", infoPath)
	err = ioutil.WriteFile(infoPath, b, 0o755)
	if err != nil {
		return errs.Wrap(err, "Failed to write info.json.")
	}

	err = fileutils.CopyFile(infoPath, filepath.Join(outputPath, relVersionedPath, filepath.Base(infoPath)))
	if err != nil {
		return errs.Wrap(err, "Could not copy info.json file")
	}

	return nil
}

func createInstaller(buildPath, outputPath, channel, platform string) error {
	installer := filepath.Join(buildPath, "state-installer"+osutils.ExeExtension)
	if !fileutils.FileExists(installer) {
		return errs.New("state-installer does not exist in build dir")
	}

	archive, archiveExt := archiveMeta()
	relArchivePath := filepath.Join(channel, platform, "state-installer"+archiveExt)
	archivePath := filepath.Join(outputPath, relArchivePath)

	// Remove archive path if it already exists
	_ = os.Remove(archivePath)
	// Create main archive
	fmt.Printf("Creating %s\n", archivePath)
	err := archive.Archive([]string{installer}, archivePath)
	if err != nil {
		return errs.Wrap(err, "Archiving failed")
	}

	return nil
}

func run() error {
	var (
		binDir        = defaultBuildDir
		inDir         = defaultInputDir
		outDir        = defaultOutputDir
		platform      = fetchPlatform()
		branch        = constants.BranchName
		version       = constants.Version
		versionNumber = constants.VersionNumber
	)

	flag.StringVar(&outDir, "o", outDir, "Override directory to output archive to.")
	flag.StringVar(
		&platform, "platform", platform,
		"Target platform in the form OS-ARCH. Defaults to running os/arch or the combination of the environment variables GOOS and GOARCH if both are set.",
	)
	flag.StringVar(&branch, "b", branch, "Override target branch. (Branch to receive update.)")
	flag.StringVar(&version, "v", version, "Override version number for this update.")
	flag.Parse()

	if err := fileutils.MkdirUnlessExists(outDir); err != nil {
		return err
	}

	if err := createUpdate(outDir, branch, version, versionNumber, platform, inDir); err != nil {
		return err
	}

	if err := createInstaller(binDir, outDir, branch, platform); err != nil {
		return err
	}

	return nil
}
