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

var outputDirFlag, platformFlag, branchFlag, versionFlag *string

func printUsage() {
	fmt.Println("")
	fmt.Println("[-o outputDir] [-b branchOverride] [-v versionOverride] [--platform platformOverride] <installer> <binaries>...")
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

func createUpdate(targetPath string, channel, version, platform string, installerPath string, binaries []string) error {
	relChannelPath := filepath.Join(channel, platform)
	relVersionedPath := filepath.Join(channel, version, platform)
	os.MkdirAll(filepath.Join(targetPath, relChannelPath), 0755)
	os.MkdirAll(filepath.Join(targetPath, relVersionedPath), 0755)

	// Copy files to a temporary directory that we can create the archive from
	tempDir, err := ioutil.TempDir("", "cli-update-generator")
	if err != nil {
		return errs.Wrap(err, "Could not create temp dir")
	}
	defer os.RemoveAll(tempDir)

	// Todo The archiver package we are using, creates an archive with a toplevel directory, so we need to give it a deterministic name ("root")
	tempDir = filepath.Join(tempDir, constants.ToplevelInstallArchiveDir)

	// copy installer to temp dir
	err = copyFileToDir(installerPath, tempDir, true)
	if err != nil {
		return errs.Wrap(err, "Failed to copy installer.")
	}

	// copy binary files to binary temp dir
	binTempDir := filepath.Join(tempDir, "bin")
	err = os.MkdirAll(binTempDir, 0755)
	if err != nil {
		return errs.Wrap(err, "Could not create temp binary dir")
	}

	for _, bf := range binaries {
		err := copyFileToDir(bf, binTempDir, true)
		if err != nil {
			return errs.Wrap(err, "Failed to copy binary file %s", bf)
		}
	}

	archive, archiveExt := archiveMeta()
	relArchivePath := filepath.Join(relVersionedPath, fmt.Sprintf("state-%s-%s%s", platform, version, archiveExt))
	archivePath := filepath.Join(targetPath, relArchivePath)

	// Remove archive path if it already exists
	_ = os.Remove(archivePath)
	// Create main archive
	fmt.Printf("Creating %s\n", archivePath)
	err = archive.Archive([]string{tempDir}, archivePath)
	if err != nil {
		return errs.Wrap(err, "Archiving failed")
	}

	up := updater.NewAvailableUpdate(version, channel, platform, filepath.ToSlash(relArchivePath), generateSha256(archivePath))
	b, err := json.MarshalIndent(up, "", "    ")
	if err != nil {
		return errs.Wrap(err, "Failed to marshal AvailableUpdate information.")
	}

	infoPath := filepath.Join(targetPath, relChannelPath, "info.json")
	fmt.Printf("Creating %s\n", infoPath)
	err = ioutil.WriteFile(infoPath, b, 0755)
	if err != nil {
		return errs.Wrap(err, "Failed to write info.json.")
	}

	return copyFileToDir(infoPath, filepath.Join(targetPath, relVersionedPath), false)
}

func run() error {
	flag.Parse()
	if flag.NArg() < 1 && !condition.InTest() {
		flag.Usage()
		printUsage()
		exit(0)
	}

	installerPath := flag.Arg(0)

	binaries := flag.Args()[1:]

	branch := constants.BranchName
	if branchFlag != nil && *branchFlag != "" {
		branch = *branchFlag
	}

	platform := *platformFlag

	version := *versionFlag

	targetDir := *outputDirFlag
	os.MkdirAll(targetDir, 0755)

	return createUpdate(targetDir, branch, version, platform, installerPath, binaries)
}
