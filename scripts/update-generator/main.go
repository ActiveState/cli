package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ActiveState/cli/internal/constants"

	"github.com/pkg/errors"
)

var appPath, version, genDir, defaultPlatform string

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

type gzReader struct {
	z, r io.ReadCloser
}

func (g *gzReader) Read(p []byte) (int, error) {
	return g.z.Read(p)
}

func (g *gzReader) Close() error {
	g.z.Close()
	return g.r.Close()
}

func newGzReader(r io.ReadCloser) io.ReadCloser {
	var err error
	g := new(gzReader)
	g.r = r
	g.z, err = gzip.NewReader(r)
	if err != nil {
		panic(err)
	}
	return g
}

func createUpdate(path string, platform string) {
	t := time.Now().Format("2006-01-02_15-04-05")
	archive := t + "--" + constants.BuildNumber + "--" + constants.RevisionHash

	os.MkdirAll(filepath.Join(genDir, constants.BranchName, version), 0755)
	os.MkdirAll(filepath.Join(genDir, constants.BranchName, version, archive), 0755)

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
	gzPath := filepath.Join(genDir, constants.BranchName, version, platform+".gz")
	err = ioutil.WriteFile(gzPath, buf.Bytes(), 0755)
	if err != nil {
		panic(errors.Wrapf(err,
			"Errored writing gzipped buffer to file"))
	}

	// Store archived version
	gzArchivePath := filepath.Join(genDir, constants.BranchName, version, archive, platform+".gz")
	err = ioutil.WriteFile(gzArchivePath, buf.Bytes(), 0755)
	if err != nil {
		panic(errors.Wrapf(err,
			"Errored writing gzipped buffer to file"))
	}

	c := current{Version: version, Sha256: generateSha256(gzPath)}
	b, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		fmt.Println("error:", err)
	}
	err = ioutil.WriteFile(filepath.Join(genDir, constants.BranchName, platform+".json"), b, 0755)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(filepath.Join(genDir, constants.BranchName, version, platform+".json"), b, 0755)
	if err != nil {
		panic(err)
	}
}

func printUsage() {
	fmt.Println("")
	fmt.Println("Positional arguments:")
	fmt.Println("\tSingle platform: go-selfupdate myapp 1.2")
	fmt.Println("\tCross platform: go-selfupdate /tmp/mybinares/ 1.2")
}

func createBuildDir() {
	os.MkdirAll(genDir, 0755)
}

func main() {
	if flag.Lookup("test.v") == nil {
		run()
	}
}

func run() {
	outputDirFlag := flag.String("o", "public", "Output directory for writing updates")

	goos := os.Getenv("GOOS")
	goarch := os.Getenv("GOARCH")
	if goos != "" && goarch != "" {
		defaultPlatform = goos + "-" + goarch
	} else {
		defaultPlatform = runtime.GOOS + "-" + runtime.GOARCH
	}
	platformFlag := flag.String("platform", defaultPlatform,
		"Target platform in the form OS-ARCH. Defaults to running os/arch or the combination of the environment variables GOOS and GOARCH if both are set.")

	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
		printUsage()
		os.Exit(0)
	}

	platform := *platformFlag

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
