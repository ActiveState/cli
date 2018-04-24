package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"

	"github.com/ActiveState/cli/internal/artifact"
	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/environment"
	"github.com/ActiveState/cli/internal/fileutils"

	"github.com/mholt/archiver"
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
	distro("linux", "x86_64", false)
	distro("macos", "x86_64", false)
	distro("windows", "x86_64", false)
	distro("linux", "x86_64", true)
	distro("macos", "x86_64", true)
	distro("windows", "x86_64", true)
}

func distro(OS string, arch string, isForTests bool) {
	var platform = fmt.Sprintf("%s-%s", OS, arch)

	// Create main distro
	fmt.Println("Creating main distro for " + platform)
	distro := []*Distribution{}

	var targetDistPath string
	if isForTests {
		targetDistPath = path.Join(environment.GetRootPathUnsafe(), "test", "distro", platform)
	} else {
		targetDistPath = path.Join(environment.GetRootPathUnsafe(), "public", "distro", platform)
	}

	os.MkdirAll(targetDistPath, 0777)

	distro = run("go", OS, distro, isForTests)

	distrob, err := json.Marshal(distro)
	if err != nil {
		log.Fatalf("JSON encoding failed: %s", err.Error())
	}

	fmt.Printf("Saving distro to %s", path.Join(targetDistPath, "distribution.json"))
	ioutil.WriteFile(path.Join(targetDistPath, "distribution.json"), distrob, 0666)
}

func run(language string, OS string, distro []*Distribution, isForTests bool) []*Distribution {
	subpath := ""
	if isForTests {
		subpath = "test"
	}

	sourceDistPath := path.Join(environment.GetRootPathUnsafe(), "scripts", "artifact-generator",
		"source", "vendor", subpath, language, "distribution", OS)
	sourceArtifactPath := path.Join(environment.GetRootPathUnsafe(), "scripts", "artifact-generator",
		"source", "vendor", subpath, language, "packages")

	targetArtifactPathRelative := path.Join("distro", "artifacts")
	targetArtifactPath := path.Join(environment.GetRootPathUnsafe(), "public", targetArtifactPathRelative)
	if isForTests {
		targetArtifactPath = path.Join(environment.GetRootPathUnsafe(), "test", targetArtifactPathRelative)
	}

	os.MkdirAll(targetArtifactPath, 0777)

	var packages []*Package
	switch language {
	case "go":
		packages = getPackagePathsGo(sourceArtifactPath)
	default:
		log.Fatalf("Unsupported language: %s", language)
	}

	languageArtifact := createArtifact(language, sourceDistPath, "language", targetArtifactPath, targetArtifactPathRelative)
	distro = append(distro, languageArtifact)

	for _, pkg := range packages {
		packageArtifact := createArtifact(pkg.Name, pkg.AbsolutePath, "package", targetArtifactPath, targetArtifactPathRelative)
		packageArtifact.Parent = languageArtifact.Hash
		distro = append(distro, packageArtifact)
	}

	return distro
}

func createArtifact(name string, srcPath string, kind string, targetPath string, downloadPath string) *Distribution {
	fmt.Printf("Creating artifact for %s: %s (%s)\n", kind, name, srcPath)

	artf := &artifact.Meta{
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
	artifactSource := path.Join(os.TempDir(), constants.ArtifactFile)
	ioutil.WriteFile(artifactSource, artfb, os.ModePerm)

	// Add source files
	files, err := ioutil.ReadDir(srcPath)
	if err != nil {
		log.Fatalf("Cannot walk source dir: %s", err.Error())
	}
	source := []string{artifactSource}
	for _, file := range files {
		source = append(source, path.Join(srcPath, file.Name()))
	}

	target := path.Join(targetPath, "artifact.tar.gz")

	fmt.Printf(" \\- Writing interim file: %s\n", target)
	err = archiver.TarGz.Make(target, source)
	if err != nil {
		log.Fatalf("Archive creation failed: %s", err.Error())
	}

	hash, fail := fileutils.Hash(target)
	if fail != nil {
		log.Fatal(fail.Error())
	}
	realTarget := path.Join(targetPath, hash+".tar.gz")

	fmt.Printf("  - Moving file to: %s\n", realTarget)
	err = os.Rename(target, realTarget)
	if err != nil {
		log.Fatalf("Could not move file from %s to %s", target, realTarget)
	}

	return &Distribution{
		Hash:     hash,
		Download: constants.APIArtifactURL + downloadPath + "/" + hash + ".tar.gz",
	}
}

func getPackagePathsGo(sourcePath string) []*Package {
	gobin := "go"
	goroot := os.Getenv("GOROOT")
	if goroot != "" {
		gobin = path.Join(goroot, "bin", "go")
	}
	cmd := exec.Command(gobin, "list", "-e", "all")
	cmd.Env = []string{"GOPATH=" + sourcePath}

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Could not run `go list`: %s, output received: %s", err.Error(), output)
	}

	resultPaths := []*Package{}
	relativePaths := strings.Split(string(output), "\n")

	sort.Sort(byLengthSorter(relativePaths))

	root := environment.GetRootPathUnsafe()

	for _, destPath := range relativePaths {
		if destPath == "" || strings.Contains(destPath, "vendor") || strings.Contains(destPath, root) || strings.Contains(destPath, ".git") {
			continue
		}

		if _, err := os.Stat(path.Join(sourcePath, "src", destPath)); os.IsNotExist(err) {
			continue
		}

		var exists bool
		for _, p := range resultPaths {
			if len(destPath) >= len(p.Name) && destPath[0:len(p.Name)] == p.Name {
				exists = true
				break
			}
		}

		if exists {
			continue
		}

		resultPaths = append(resultPaths, &Package{destPath, path.Join(sourcePath, "src", destPath)})
	}

	return resultPaths
}
