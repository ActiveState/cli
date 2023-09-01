package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

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
	infoFileName     = "info.json"
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
	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatalln(err)
	}
	if _, err := hasher.Write(b); err != nil {
		log.Fatalln(err)
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

func archiveMeta() (archiveMethod archiver.Archiver, ext string) {
	if runtime.GOOS == "windows" {
		return archiver.NewZip(), ".zip"
	}
	return archiver.NewTarGz(), ".tar.gz"
}

func systemSHA256Sum(file string) string {
	cmdText := "sha256sum"
	var cmdArgs []string

	if runtime.GOOS == "darwin" {
		cmdText = "shasum"
		cmdArgs = []string{"-a", "256"}
	}

	cmdArgs = append(cmdArgs, file)

	cmd := exec.Command(cmdText, cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("error collecting sha sum: gather combined output: %v\n", err)
		return ""
	}

	rawText := string(out)
	sum, _, ok := strings.Cut(rawText, " ")
	if !ok {
		fmt.Printf("error collecting sha sum: cannot find sum in %q\n", rawText)
		return ""
	}

	return sum
}

func createUpdate(outputPath, channel, version, platform, target string) error {
	relChannelPath := filepath.Join(channel, platform)
	relVersionedPath := filepath.Join(channel, version, platform)
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

	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		fmt.Printf("TmpDir creation failed: %v\n", err)
	}
	if err := archiver.Unarchive(archivePath, tmpDir); err != nil {
		fmt.Printf("Unarchiving failed: %v\n", err)
	}

	avUpdate := updater.NewAvailableUpdate(channel, version, platform, filepath.ToSlash(relArchivePath), generateSha256(archivePath), "")
	b, err := json.MarshalIndent(avUpdate, "", "    ")
	if err != nil {
		return errs.Wrap(err, "Failed to marshal AvailableUpdate information.")
	}

	infoPath := filepath.Join(outputPath, relChannelPath, infoFileName)
	fmt.Printf("Creating %s\n", infoPath)
	err = os.WriteFile(infoPath, b, 0o755)
	if err != nil {
		return errs.Wrap(err, "Failed to write info file (%s).", infoPath)
	}

	copyInfoPath := filepath.Join(outputPath, relVersionedPath, filepath.Base(infoPath))
	fmt.Printf("Creating copy of info file as %s\n", copyInfoPath)
	err = fileutils.CopyFile(infoPath, copyInfoPath)
	if err != nil {
		return errs.Wrap(err, "Could not copy info file to (%s).", copyInfoPath)
	}

	fmt.Printf("Generated SHA sum: %s\n", avUpdate.Sha256)

	systemSum := systemSHA256Sum(archivePath)
	fmt.Printf("System calculated SHA sum: %s\n", systemSum)

	return nil
}

func createInstaller(buildPath, outputPath, channel, platform string) error {
	installer := filepath.Join(buildPath, "state-installer"+osutils.ExeExt)
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
		binDir   = defaultBuildDir
		inDir    = defaultInputDir
		outDir   = defaultOutputDir
		platform = fetchPlatform()
		branch   = constants.BranchName
		version  = constants.Version
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

	if err := createUpdate(outDir, branch, version, platform, inDir); err != nil {
		return err
	}

	if err := createInstaller(binDir, outDir, branch, platform); err != nil {
		return err
	}

	return nil
}
