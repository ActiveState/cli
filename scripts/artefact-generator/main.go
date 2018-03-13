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
	"github.com/mholt/archiver"

	"github.com/ActiveState/ActiveState-CLI/internal/artefact"
)

// Distribution reflects the data contained in the distribution.json file
type Distribution struct {
	Hash     string
	Parent   string
	Download string
}

// Package is used to iterate through packages found, before they are turned into artefacts
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

	sourceDistPath := filepath.Join(environment.GetRootPathUnsafe(), "scripts", "artefact-generator",
		"source", language, "distribution", OS)
	sourceArtefactPath := filepath.Join(environment.GetRootPathUnsafe(), "scripts", "artefact-generator", "source", language, "packages")

	targetDistPath := filepath.Join(environment.GetRootPathUnsafe(), "public", "distro", language, platform)
	targetArtefactPathRelative := filepath.Join("distro", language, "artefacts")
	targetArtefactPath := filepath.Join(environment.GetRootPathUnsafe(), "public", targetArtefactPathRelative)

	os.MkdirAll(targetDistPath, os.ModePerm)
	os.MkdirAll(targetArtefactPath, os.ModePerm)

	var packages []*Package
	switch language {
	case "go":
		packages = getPackagePathsGo(sourceArtefactPath)
	default:
		log.Fatalf("Unsupported language: %s", language)
	}

	distro := []*Distribution{}
	languageArtefact := createArtefact(language, sourceDistPath, "language", targetArtefactPath, targetArtefactPathRelative)
	distro = append(distro, languageArtefact)

	for _, pkg := range packages {
		packageArtefact := createArtefact(pkg.Name, pkg.AbsolutePath, "package", targetArtefactPath, targetArtefactPathRelative)
		packageArtefact.Parent = languageArtefact.Hash
		distro = append(distro, packageArtefact)
	}

	distrob, err := json.Marshal(distro)
	if err != nil {
		log.Fatalf("JSON encoding failed: %s", err.Error())
	}
	ioutil.WriteFile(filepath.Join(targetDistPath, "distribution.json"), distrob, os.ModePerm)
}

func createArtefact(name string, path string, kind string, targetPath string, downloadPath string) *Distribution {
	fmt.Printf("Creating artefact for %s: %s (%s)\n", kind, name, path)

	artf := &artefact.Artefact{
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
	artefactSource := filepath.Join(os.TempDir(), "artefact.json")
	ioutil.WriteFile(artefactSource, artfb, os.ModePerm)

	// Add source files
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatalf("Cannot walk source dir: %s", err.Error())
	}
	source := []string{artefactSource}
	for _, file := range files {
		source = append(source, filepath.Join(path, file.Name()))
	}

	target := filepath.Join(targetPath, "artefact.tar.gz")

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
		Download: constants.APIArtefactURL + downloadPath + hash + ".tar.gz",
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
