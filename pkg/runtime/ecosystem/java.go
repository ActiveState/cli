package ecosystem

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"

	"github.com/ActiveState/cli/pkg/buildplan"
)

const libDir = "lib"

type Java struct {
	runtimeDir string
}

func (e *Java) Init(runtimePath string, buildplan *buildplan.BuildPlan) error {
	e.runtimeDir = runtimePath
	err := fileutils.MkdirUnlessExists(filepath.Join(e.runtimeDir, libDir))
	if err != nil {
		return errs.Wrap(err, "Unable to create runtime lib directory")
	}
	return nil
}

func (e *Java) Namespaces() []string {
	return []string{"language/java"}
}

func (e *Java) Add(artifact *buildplan.Artifact, artifactSrcPath string) ([]string, error) {
	installedFiles := []string{}
	files, err := fileutils.ListDir(artifactSrcPath, false)
	if err != nil {
		return nil, errs.Wrap(err, "Unable to read artifact source directory")
	}
	for _, file := range files {
		if file.Name() == "runtime.json" {
			err = e.injectClasspath(file.AbsolutePath())
			if err != nil {
				return nil, errs.Wrap(err, "Unable to add CLASSPATH to runtime.json")
			}
			continue
		}
		if !strings.HasSuffix(file.Name(), ".jar") {
			continue
		}
		relativeInstalledFile := filepath.Join(libDir, file.Name())
		installedFile := filepath.Join(e.runtimeDir, relativeInstalledFile)
		err = fileutils.CopyFile(file.AbsolutePath(), installedFile)
		if err != nil {
			return nil, errs.Wrap(err, "Unable to copy artifact jar into runtime lib directory")
		}
		installedFiles = append(installedFiles, relativeInstalledFile)
	}
	return installedFiles, nil
}

func (e *Java) Remove(artifact *buildplan.Artifact) error {
	return nil // TODO: CP-956
}

func (e *Java) Apply() error {
	return nil
}

func (e *Java) injectClasspath(runtimeJson string) error {
	bytes, err := fileutils.ReadFile(runtimeJson)
	if err != nil {
		return errs.Wrap(err, "Unable to read runtime.json")
	}

	m := make(map[string]interface{})
	err = json.Unmarshal(bytes, &m)
	if err != nil {
		return errs.Wrap(err, "Unable to unmarshal runtime.json")
	}

	classpathEnv := map[string]interface{}{
		"env_name":  "CLASSPATH",
		"values":    []string{"${INSTALLDIR}/lib"},
		"join":      "prepend",
		"inherit":   true,
		"separator": ":",
	}

	classpathExists := false
	if _, exists := m["env"]; !exists {
		m["env"] = make([]map[string]interface{}, 0)
	}
	envList := m["env"].([]interface{})
	for _, envInterface := range envList {
		env := envInterface.(map[string]interface{})
		if env["env_name"] == "CLASSPATH" {
			classpathExists = true
			break
		}
	}

	if !classpathExists {
		envList = append(envList, classpathEnv)
		m["env"] = envList

		bytes, err = json.Marshal(m)
		if err != nil {
			return errs.Wrap(err, "Unable to marshal new runtime.json")
		}

		err = fileutils.WriteFile(runtimeJson, bytes)
		if err != nil {
			return errs.Wrap(err, "Unable to write new runtime.json")
		}
	}

	return nil
}
