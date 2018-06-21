package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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

	distro = []*Distribution{}
	distro = run("go", OS, distro, isForTests)
	distro = run("python2", OS, distro, isForTests)
	distro = run("python3", OS, distro, isForTests)
	distro = run("perl", OS, distro, isForTests)

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
	var relocate string
	switch language {
	case "go":
		packages = getPackagePathsGo(sourceArtifactPath)
	case "python3":
		packages = getPackagePaths(sourceArtifactPath)
		relocate = getRelocatePython(sourceDistPath, "3.5")
	case "python2":
		packages = getPackagePaths(sourceArtifactPath)
		relocate = getRelocatePython(sourceDistPath, "2.7")
	case "perl":
		packages = getPackagePaths(sourceArtifactPath)
	default:
		log.Fatalf("Unsupported language: %s", language)
	}

	languageArtifact := createArtifact(language, sourceDistPath, "language", targetArtifactPath, targetArtifactPathRelative, relocate)
	distro = append(distro, languageArtifact)

	for _, pkg := range packages {
		packageArtifact := createArtifact(pkg.Name, pkg.AbsolutePath, "package", targetArtifactPath, targetArtifactPathRelative, relocate)
		packageArtifact.Parent = languageArtifact.Hash
		distro = append(distro, packageArtifact)
	}

	return distro
}

func createArtifact(name string, srcPath string, kind string, targetPath string, downloadPath string, relocate string) *Distribution {
	fmt.Printf("Creating artifact for %s: %s (%s)\n", kind, name, srcPath)

	artf := &artifact.Meta{
		Name:     name,
		Type:     kind,
		Version:  "0.0.1", // versions arent supported by this implementation
		Relocate: relocate,
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

func getPackagePaths(sourcePath string) []*Package {
	files, err := ioutil.ReadDir(sourcePath)
	if err != nil {
		panic(err.Error())
	}

	resultPaths := []*Package{}
	for _, f := range files {
		filename := f.Name()
		packageName := strings.TrimSuffix(filename, filepath.Ext(filename))
		resultPaths = append(resultPaths, &Package{packageName, filepath.Join(sourcePath, filename)})
	}

	return resultPaths
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

func getRelocatePython(sourceDistPath string, version string) string {
	var path = filepath.Join(sourceDistPath, "lib", "python"+version, "activestate.py")
	if !fileutils.FileExists(path) {
		path = filepath.Join(sourceDistPath, "Lib", "activestate.py") // Python 2.7 on Windows
	}
	if !fileutils.FileExists(path) {
		return ""
	}

	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var relocate string
	var scanner = bufio.NewScanner(file)
	var nextLine = false
	for scanner.Scan() {
		var line = scanner.Text()
		if nextLine {
			relocate = line[strings.Index(line, "'")+1:]
			relocate = relocate[0:strings.Index(relocate, "'")]
			break
		}
		nextLine = strings.Contains(line, "# Prefix to which extensions were built")
	}

	return relocate
}
