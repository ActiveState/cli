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
	"time"

	"github.com/ActiveState/archiver"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/pkg/errors"
)

var exit = os.Exit

var appPath, version, genDir, defaultPlatform, branch string

var outputDirFlag, platformFlag, branchFlag *string

type current struct {
	Version string
	Sha256  string
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
	t := time.Now().Format("2006-01-02_15-04-05")
	archiveName := t + "--" + constants.BuildNumber + "--" + constants.RevisionHash

	os.MkdirAll(filepath.Join(genDir, branch, version), 0755)
	os.MkdirAll(filepath.Join(genDir, branch, version, archiveName), 0755)

	// Prepare the archiver
	var ext, binExt string
	var archive archiver.Archiver
	if runtime.GOOS == "windows" {
		archive = archiver.NewZip()
		ext = ".zip"
		binExt = ".exe"
	} else {
		archive = archiver.NewTarGz()
		ext = ".tar.gz"
	}

	// Copy to a temp path so we can use a custom filename
	tempDir, err := ioutil.TempDir("", "cli-update-generator")
	if err != nil {
		panic(errors.Wrap(err, "Could not create temp dir"))
	}
	tempPath := filepath.Join(tempDir, platform+binExt)
	fail := fileutils.CopyFile(path, tempPath)
	if fail != nil {
		panic(errors.Wrap(fail.ToError(), "Copy failed"))
	}

	targetDir := filepath.Join(genDir, branch, version)
	targetPath := filepath.Join(targetDir, platform+ext)
	targetArchivePath := filepath.Join(targetDir, archiveName, platform+ext)

	// Remove target files if it already exists
	if fileutils.FileExists(targetPath) {
		err = os.Remove(targetPath)
		if err != nil {
			panic(errors.Wrap(err, "Could not remove target path"))
		}
	}
	if fileutils.FileExists(targetArchivePath) {
		err = os.Remove(targetArchivePath)
		if err != nil {
			panic(errors.Wrap(err, "Could not remove target archive path"))
		}
	}

	// Create main archive
	fmt.Printf("Creating %s\n", targetPath)
	err = archive.Archive([]string{tempPath}, targetPath)
	if err != nil {
		panic(errors.Wrap(err, "Archiving failed"))
	}

	// Create historical archive
	fmt.Printf("Creating %s\n", targetArchivePath)
	fail = fileutils.CopyFile(targetPath, targetArchivePath)
	if fail != nil {
		panic(errors.Wrap(fail.ToError(), "Copy failed"))
	}

	c := current{Version: version, Sha256: generateSha256(targetPath)}
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

	jsonPath = filepath.Join(genDir, branch, version, platform+".json")
	fmt.Printf("Creating %s\n", jsonPath)
	err = ioutil.WriteFile(jsonPath, b, 0755)
	if err != nil {
		panic(err)
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
	if flag.Lookup("test.v") == nil {
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
	goos := os.Getenv("GOOS")
	goarch := os.Getenv("GOARCH")
	if goos != "" && goarch != "" {
		defaultPlatform = goos + "-" + goarch
	} else {
		defaultPlatform = runtime.GOOS + "-" + runtime.GOARCH
	}

	flag.Parse()
	if flag.NArg() < 1 && flag.Lookup("test.v") == nil {
		flag.Usage()
		printUsage()
		exit(0)
	}

	a := flag.Args()
	_ = a

	if appPath == "" {
		appPath = flag.Arg(0)
	}

	if version == "" {
		if flag.Arg(1) == "" {
			version = constants.Version
		} else {
			version = flag.Arg(1)
		}
	}

	if branchFlag != nil && *branchFlag != "" {
		branch = *branchFlag
	} else {
		branch = constants.BranchName
	}

	var platform string
	if platformFlag != nil && *platformFlag != "" {
		platform = *platformFlag
	} else {
		platform = defaultPlatform
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
