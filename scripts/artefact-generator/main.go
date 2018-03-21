package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ActiveState/ActiveState-CLI/internal/fileutils"

	"github.com/ActiveState/ActiveState-CLI/internal/constants"
	"github.com/ActiveState/ActiveState-CLI/internal/environment"
	"github.com/ActiveState/sysinfo"
	"github.com/mholt/archiver"

	"github.com/ActiveState/ActiveState-CLI/internal/artefact"
)

// OS is uppercase cause os is taken
var OS = strings.ToLower(sysinfo.OS().String())
var arch = strings.ToLower(sysinfo.Architecture().String())
var platform = fmt.Sprintf("%s-%s", OS, arch)

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
	// Create main distro
	fmt.Println("Creating main distro")
	distro := []*Distribution{}

	targetDistPath := filepath.Join(environment.GetRootPathUnsafe(), "public", "distro", platform)
	os.MkdirAll(targetDistPath, os.ModePerm)

	distro = run("go", distro, false)

	distrob, err := json.Marshal(distro)
	if err != nil {
		log.Fatalf("JSON encoding failed: %s", err.Error())
	}

	fmt.Printf("Saving distro to %s", filepath.Join(targetDistPath, "distribution.json"))
	ioutil.WriteFile(filepath.Join(targetDistPath, "distribution.json"), distrob, os.ModePerm)

	// Create test distro
	fmt.Println("Creating test distro")
	distro = []*Distribution{}

	targetDistPath = filepath.Join(environment.GetRootPathUnsafe(), "test", "distro")
	os.MkdirAll(targetDistPath, os.ModePerm)

	distro = run("go", distro, true)

	distrob, err = json.Marshal(distro)
	if err != nil {
		log.Fatalf("JSON encoding failed: %s", err.Error())
	}

	fmt.Printf("Saving distro to %s", filepath.Join(targetDistPath, "distribution.json"))
	ioutil.WriteFile(filepath.Join(targetDistPath, "distribution.json"), distrob, os.ModePerm)
}

func run(language string, distro []*Distribution, isForTests bool) []*Distribution {
	subpath := ""
	if isForTests {
		subpath = "test"
	}

	sourceDistPath := filepath.Join(environment.GetRootPathUnsafe(), "scripts", "artefact-generator",
		"source", "vendor", subpath, language, "distribution", OS)
	sourceArtefactPath := filepath.Join(environment.GetRootPathUnsafe(), "scripts", "artefact-generator",
		"source", "vendor", subpath, language, "packages")

	targetArtefactPathRelative := filepath.Join("distro", "artefacts")
	targetArtefactPath := filepath.Join(environment.GetRootPathUnsafe(), "public", targetArtefactPathRelative)
	if isForTests {
		targetArtefactPath = filepath.Join(environment.GetRootPathUnsafe(), "test", targetArtefactPathRelative)
	}

	os.MkdirAll(targetArtefactPath, os.ModePerm)

	var packages []*Package
	switch language {
	case "go":
		packages = getPackagePathsGo(sourceArtefactPath)
	default:
		log.Fatalf("Unsupported language: %s", language)
	}

	languageArtefact := createArtefact(language, sourceDistPath, "language", targetArtefactPath, targetArtefactPathRelative)
	distro = append(distro, languageArtefact)

	for _, pkg := range packages {
		packageArtefact := createArtefact(pkg.Name, pkg.AbsolutePath, "package", targetArtefactPath, targetArtefactPathRelative)
		packageArtefact.Parent = languageArtefact.Hash
		distro = append(distro, packageArtefact)
	}

	return distro
}

func createArtefact(name string, path string, kind string, targetPath string, downloadPath string) *Distribution {
	fmt.Printf("Creating artefact for %s: %s (%s)\n", kind, name, path)

	artf := &artefact.Meta{
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
	artefactSource := filepath.Join(os.TempDir(), constants.ArtefactFile)
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

	hash, fail := fileutils.Hash(target)
	if fail != nil {
		log.Fatal(fail.Error())
	}
	realTarget := filepath.Join(targetPath, hash+".tar.gz")

	fmt.Printf("  - Moving file to: %s\n", realTarget)
	err = os.Rename(target, realTarget)
	if err != nil {
		log.Fatalf("Could not move file from %s to %s", target, realTarget)
	}

	return &Distribution{
		Hash:     hash,
		Download: constants.APIArtefactURL + downloadPath + "/" + hash + ".tar.gz",
	}
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
			if len(path) >= len(p.Name) && path[0:len(p.Name)] == p.Name {
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
