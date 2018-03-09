package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	"github.com/ActiveState/sysinfo"
	logging "github.com/hhkbp2/go-logging"
	"github.com/mholt/archiver"

	"github.com/ActiveState/ActiveState-CLI/internal/artifact"
)

// Distribution reflects the data contained in the distribution.json file
type Distribution struct {
	Hash     string
	Parent   string
	Download string
}

// Package is used to iterate through packages found, before they are turned into artifacts
type Package struct {
	Name         string
	AbsolutePath string
}

type byLengthSorter []string

func (s byLengthSorter) Len() int {
	return len(s)
}
func (s byLengthSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byLengthSorter) Less(i, j int) bool {
	return len(s[i]) < len(s[j])
}

func main() {
	run("go")
}

func run(language string) {
	OS := strings.ToLower(sysinfo.OS().String())
	arch := strings.ToLower(sysinfo.Architecture().String())
	platform := fmt.Sprintf("%s-%s", OS, arch)

	sourceDistPath := filepath.Join(environment.GetRootPathUnsafe(), "scripts", "artifact-generator",
		"source", language, "distribution", strings.ToLower(OS))
	sourceArtifactPath := filepath.Join(environment.GetRootPathUnsafe(), "scripts", "artifact-generator", "source", language, "packages")

	targetDistPath := filepath.Join(environment.GetRootPathUnsafe(), "public", "distro", language, platform)
	targetArtifactPathRelative := filepath.Join("distro", language, "artifacts")
	targetArtifactPath := filepath.Join(environment.GetRootPathUnsafe(), "public", targetArtifactPathRelative)

	os.MkdirAll(targetDistPath, os.ModePerm)
	os.MkdirAll(targetArtifactPath, os.ModePerm)

	var packages []*Package
	switch language {
	case "go":
		packages = getPackagePathsGo(sourceArtifactPath)
	default:
		logging.Fatalf("Unsupported language: %s", language)
	}

	distro := []*Distribution{}
	languageArtifact := createArtifact(language, sourceDistPath, "language", targetArtifactPath, targetArtifactPathRelative)
	distro = append(distro, languageArtifact)

	for _, pkg := range packages {
		packageArtifact := createArtifact(pkg.Name, pkg.AbsolutePath, "package", targetArtifactPath, targetArtifactPathRelative)
		packageArtifact.Parent = languageArtifact.Hash
		distro = append(distro, packageArtifact)
	}

	distrob, err := json.Marshal(distro)
	if err != nil {
		log.Fatalf("JSON encoding failed: %s", err.Error())
	}
	ioutil.WriteFile(filepath.Join(targetDistPath, "distribution.json"), distrob, os.ModePerm)
}

func createArtifact(name string, path string, kind string, targetPath string, downloadPath string) *Distribution {
	fmt.Printf("Creating artifact for %s: %s (%s)\n", kind, name, path)

	artf := &artifact.Artifact{
		Name:     name,
		Type:     kind,
		Version:  "0.0.1", // versions arent supported by this implementation
		Relocate: "",
		Binaries: []string{},
	}
	artfb, err := json.Marshal(artf)
	if err != nil {
		log.Fatalf("JSON encoding failed: %s", err.Error())
	}
	artifactSource := filepath.Join(os.TempDir(), "artifact.json")
	ioutil.WriteFile(artifactSource, artfb, os.ModePerm)

	// Add source files
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatalf("Cannot walk source dir: %s", err.Error())
	}
	source := []string{artifactSource}
	for _, file := range files {
		source = append(source, filepath.Join(path, file.Name()))
	}

	target := filepath.Join(targetPath, "artifact.tar.gz")

	fmt.Printf(" \\- Writing interim file: %s\n", target)
	err = archiver.TarGz.Make(target, source)
	if err != nil {
		log.Fatalf("Archive creation failed: %s", err.Error())
	}

	hash := hashFromFile(target)
	realTarget := filepath.Join(targetPath, hash+".tar.gz")

	fmt.Printf("  - Moving file to: %s\n", realTarget)
	err = os.Rename(target, realTarget)
	if err != nil {
		log.Fatalf("Could not move file from %s to %s", target, realTarget)
	}

	return &Distribution{
		Hash:     hash,
		Download: constants.APIArtifactURL + downloadPath + hash + ".tar.gz",
	}
}

func hashFromFile(path string) string {
	h := sha256.New()
	b, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("Cannot read archive: %s, %s", path, err)
	}
	h.Write(b)
	sum := h.Sum(nil)
	return fmt.Sprintf("%x", sum)
}

func getPackagePathsGo(sourcePath string) []*Package {
	cmd := exec.Command("go", "list", "-e", "all")
	cmd.Env = []string{"GOPATH=" + sourcePath}

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Could not run `go list`: %s, output received: %s", err.Error(), output)
	}

	resultPaths := []*Package{}
	relativePaths := strings.Split(string(output), "\n")

	sort.Sort(byLengthSorter(relativePaths))

	root := environment.GetRootPathUnsafe()

	for _, path := range relativePaths {
		if path == "" || strings.Contains(path, "vendor") || strings.Contains(path, root) || strings.Contains(path, ".git") {
			continue
		}

		if _, err := os.Stat(filepath.Join(sourcePath, "src", path)); os.IsNotExist(err) {
			continue
		}

		var exists bool
		for _, p := range resultPaths {
			if len(path) >= len(p.AbsolutePath) && path[0:len(p.AbsolutePath)] == p.AbsolutePath {
				exists = true
				break
			}
		}

		if exists {
			continue
		}

		resultPaths = append(resultPaths, &Package{path, filepath.Join(sourcePath, "src", path)})
	}

	return resultPaths
}
