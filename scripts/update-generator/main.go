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

	"github.com/phayes/permbits"

	"github.com/ActiveState/archiver"
	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/updater"
)

var exit = os.Exit

var outputDirFlag, platformFlag, branchFlag, versionFlag, systemAppDirFlag *string

func printUsage() {
	fmt.Println("")
	fmt.Println("[-o outputDir] [-b branchOverride] [-v versionOverride] [--platform platformOverride] <directory>")
}

func main() {
	if !condition.InTest() {
		err := run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s error: %v", os.Args[0], errs.Join(err, ":"))
		}
	}
}

func init() {
	defaultPlatform := fetchPlatform()
	outputDirFlag = flag.String("o", "public", "Output directory for writing updates")
	platformFlag = flag.String("platform", defaultPlatform,
		"Target platform in the form OS-ARCH. Defaults to running os/arch or the combination of the environment variables GOOS and GOARCH if both are set.")
	branchFlag = flag.String("b", "", "Override target branch. This is the branch that will receive this update.")
	versionFlag = flag.String("v", constants.Version, "Override version number for this update.")
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

func copyFileToDir(filePath, dir string, isExecutable bool) error {
	targetPath := filepath.Join(dir, filepath.Base(filePath))
	fmt.Printf("Copying %s -> %s\n", filePath, targetPath)
	err := fileutils.CopyFile(filePath, targetPath)
	if err != nil {
		return errs.Wrap(err, "Could not copy file %s -> %s", filePath, targetPath)
	}
	if !isExecutable {
		return nil
	}
	// Permissions may be lost due to the file copy, so ensure it's still executable
	permissions, err := permbits.Stat(targetPath)
	if err != nil {
		return errs.Wrap(err, "Could not stat target file %s", targetPath)
	}
	permissions.SetUserExecute(true)
	permissions.SetGroupExecute(true)
	permissions.SetOtherExecute(true)
	err = permbits.Chmod(targetPath, permissions)
	if err != nil {
		return errs.Wrap(err, "Could not make file executable")
	}
	return nil
}

func archiveMeta() (archiveMethod archiver.Archiver, ext string) {
	if runtime.GOOS == "windows" {
		return archiver.NewZip(), ".zip"
	}
	return archiver.NewTarGz(), ".tar.gz"
}

func createUpdate(outputPath, channel, version, platform, installerPath, target string) error {
	relChannelPath := filepath.Join(channel, platform)
	relVersionedPath := filepath.Join(channel, version, platform)
	os.MkdirAll(filepath.Join(outputPath, relChannelPath), 0755)
	os.MkdirAll(filepath.Join(outputPath, relVersionedPath), 0755)

	archive, archiveExt := archiveMeta()
	relArchivePath := filepath.Join(relVersionedPath, fmt.Sprintf("state-%s-%s%s", platform, version, archiveExt))
	archivePath := filepath.Join(outputPath, relArchivePath)

	// Remove archive path if it already exists
	_ = os.Remove(archivePath)
	// Create main archive
	fmt.Printf("Creating %s\n", archivePath)
	err := archive.Archive([]string{target}, archivePath)
	if err != nil {
		return errs.Wrap(err, "Archiving failed")
	}

	up := updater.NewAvailableUpdate(version, channel, platform, filepath.ToSlash(relArchivePath), generateSha256(archivePath))
	b, err := json.MarshalIndent(up, "", "    ")
	if err != nil {
		return errs.Wrap(err, "Failed to marshal AvailableUpdate information.")
	}

	infoPath := filepath.Join(outputPath, relChannelPath, "info.json")
	fmt.Printf("Creating %s\n", infoPath)
	err = ioutil.WriteFile(infoPath, b, 0755)
	if err != nil {
		return errs.Wrap(err, "Failed to write info.json.")
	}

	return copyFileToDir(infoPath, filepath.Join(outputPath, relVersionedPath), false)
}

func run() error {
	flag.Parse()
	if flag.NArg() < 1 && !condition.InTest() {
		flag.Usage()
		printUsage()
		exit(0)
	}

	installerPath := flag.Arg(0)

	target := flag.Args()[0]

	branch := constants.BranchName
	if branchFlag != nil && *branchFlag != "" {
		branch = *branchFlag
	}

	platform := *platformFlag

	version := *versionFlag

	outputDir := *outputDirFlag
	os.MkdirAll(outputDir, 0755)

	return createUpdate(outputDir, branch, version, platform, installerPath, target)
}
