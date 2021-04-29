package main

import (
	"bytes"
	"compress/gzip"
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
	"github.com/pkg/errors"

	"github.com/ActiveState/archiver"

	"github.com/ActiveState/cli/internal/condition"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
)

var exit = os.Exit

var appPath, version, genDir, defaultPlatform, branch string

var outputDirFlag, platformFlag, branchFlag *string

type current struct {
	Version  string
	Sha256v2 string
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

func createUpdate(path string, platform string) {
	os.MkdirAll(filepath.Join(genDir, branch, version), 0755)

	// Prepare the archiver
	archive, ext, binExt := archiveMeta()

	// Copy to a temp path so we can use a custom filename
	tempDir, err := ioutil.TempDir("", "cli-update-generator")
	if err != nil {
		panic(errors.Wrap(err, "Could not create temp dir"))
	}
	tempPath := filepath.Join(tempDir, platform+binExt)
	err = fileutils.CopyFile(path, tempPath)
	if err != nil {
		panic(errors.Wrap(err, "Copy failed"))
	}

	// Permissions may be lost due to the file copy, so ensure it's still executable
	permissions, _ := permbits.Stat(tempPath)
	permissions.SetUserExecute(true)
	err = permbits.Chmod(tempPath, permissions)
	if err != nil {
		panic(errors.Wrap(err, "Could not make file executable"))
	}

	targetDir := filepath.Join(genDir, branch, version)
	targetPath := filepath.Join(targetDir, platform+ext)

	// Remove target path if it already exists
	os.Remove(targetPath)

	// Create main archive
	fmt.Printf("Creating %s\n", targetPath)
	err = archive.Archive([]string{tempPath}, targetPath)
	if err != nil {
		panic(errors.Wrap(err, "Archiving failed"))
	}

	c := current{Version: version, Sha256v2: generateSha256(targetPath)}
	b, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		fmt.Println("error:", err)
	}

	jsonPath := filepath.Join(genDir, branch, platform+".json")
	fmt.Printf("Creating %s\n", jsonPath)
	err = ioutil.WriteFile(jsonPath, b, 0755)
	if err != nil {
		panic(err)
	}

	versionPath := filepath.Join(genDir, "version.json")
	fmt.Printf("Updating version file at %s\n", versionPath)
	err = ioutil.WriteFile(versionPath, b, 0755)
	if err != nil {
		panic(err)
	}

	copy(jsonPath, filepath.Join(genDir, branch, version, platform+".json"))
}

func createGzip(path string, target string) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	f, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}

	_, err = w.Write(f)
	if err != nil {
		panic(errors.Wrapf(err,
			"Errored writing to gzip writer"))
	}
	err = w.Close() // You must close this first to flush the bytes to the buffer.
	if err != nil {
		panic(errors.Wrapf(err,
			"Errored closing gzip writter"))
	}
	err = ioutil.WriteFile(target, buf.Bytes(), 0755)
	if err != nil {
		panic(errors.Wrapf(err,
			"Errored writing gzipped buffer to file"))
	}
}

func archiveMeta() (archiveMethod archiver.Archiver, ext string, binExt string) {
	if runtime.GOOS == "windows" {
		return archiver.NewZip(), ".zip", ".exe"
	}
	return archiver.NewTarGz(), ".tar.gz", ""
}

func copy(path, target string) {
	fmt.Printf("Copying %s to %s\n", path, target)
	err := fileutils.CopyFile(path, target)
	if err != nil {
		panic(errors.Wrap(err, "Copy failed"))
	}
}

func remove(paths ...string) {
	for _, path := range paths {
		if fileutils.FileExists(path) {
			err := os.Remove(path)
			if err != nil {
				panic(errors.Wrap(err, "Could not remove path: "+path))
			}
		}
	}
}

func printUsage() {
	fmt.Println("")
	fmt.Println("[-o outputDir] [-b branchOverride] [--platform platformOverride] <appPath> [<versionOverride>]")
}

func createBuildDir() {
	os.MkdirAll(genDir, 0755)
}

func main() {
	if !condition.InTest() {
		run()
	}
}

func init() {
	outputDirFlag = flag.String("o", "public", "Output directory for writing updates")
	platformFlag = flag.String("platform", defaultPlatform,
		"Target platform in the form OS-ARCH. Defaults to running os/arch or the combination of the environment variables GOOS and GOARCH if both are set.")
	branchFlag = flag.String("b", "", "Override target branch. This is the branch that will receive this update.")
}

func run() {
	defaultPlatform = fetchPlatform()

	flag.Parse()
	if flag.NArg() < 1 && !condition.InTest() {
		flag.Usage()
		printUsage()
		exit(0)
	}

	if appPath == "" {
		appPath = flag.Arg(0)
	}

	if version == "" {
		version = flag.Arg(1)
		if flag.Arg(1) == "" {
			version = constants.Version
		}
	}

	branch = constants.BranchName
	if branchFlag != nil && *branchFlag != "" {
		branch = *branchFlag
	}

	platform := defaultPlatform
	if platformFlag != nil && *platformFlag != "" {
		platform = *platformFlag
	}

	if genDir == "" {
		genDir = *outputDirFlag
	}

	createBuildDir()

	// If dir is given create update for each file
	fi, err := os.Stat(appPath)
	if err != nil {
		panic(err)
	}

	if fi.IsDir() {
		files, err := ioutil.ReadDir(appPath)
		if err == nil {
			for _, file := range files {
				createUpdate(filepath.Join(appPath, file.Name()), file.Name())
			}
			os.Exit(0)
		}
	}

	createUpdate(appPath, platform)
}

func fetchPlatform() string {
	goos := os.Getenv("GOOS")
	goarch := os.Getenv("GOARCH")
	if goos != "" && goarch != "" {
		return goos + "-" + goarch
	}
	return runtime.GOOS + "-" + runtime.GOARCH
}
