package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/pkg/executors/execmeta"
)

const (
	envVarSeparator   = "="
	pathEnvVarPrefix  = "PATH" + envVarSeparator
	pathListSeparator = string(os.PathListSeparator)
)

type executorMeta struct {
	*execmeta.ExecMeta
	MatchingBin    string
	TransformedEnv []string
}

func newExecutorMeta(execPath string) (*executorMeta, error) {
	execDir := filepath.Dir(execPath)
	metaPath := filepath.Join(execDir, execmeta.MetaFileName)
	meta, err := execmeta.NewFromFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("cannot get execmeta from file: %w", err)
	}

	matchingBin, err := matchingBinByPath(meta.Bins, execPath)
	if err != nil {
		return nil, fmt.Errorf("cannot get matching bin by path: %w", err)
	}

	em := executorMeta{
		ExecMeta:       meta,
		MatchingBin:    matchingBin,
		TransformedEnv: transformedEnv(os.Environ(), meta.Env),
	}

	return &em, nil
}

// matchingBinByPath receives a list of binaries (from the meta file), as well
// as the path to this program. The base name of the path is used to match one
// of the binaries that will then be forwarded to as a child process.
func matchingBinByPath(bins map[string]string, path string) (string, error) {
	alias := filepath.Base(path)
	if dest, ok := bins[alias]; ok {
		return dest, nil
	}
	return "", fmt.Errorf("no matching binary by path %q", path)
}

// transformedEnv will update the current environment. Update entries are
// appended (which supersede existing entries) except for: PATH is updated with
// the update value prepended to the existing PATH.
func transformedEnv(current []string, updates []string) []string {
	for _, update := range updates {
		if strings.HasPrefix(strings.ToLower(update), strings.ToLower(pathEnvVarPrefix)) {
			pathCurrentV, pathCurrentK, ok := getEnvVar(current, pathEnvVarPrefix)
			if ok {
				current[pathCurrentK] = update + pathListSeparator + pathCurrentV
				continue
			}
		}
		current = append(current, update)
	}
	return current
}

// getEnvVar returns the value and index of the environment variable from the
// provided environment slice. The prefix should end with the environment
// variable separator.
func getEnvVar(env []string, prefix string) (string, int, bool) {
	for k, v := range env {
		if strings.HasPrefix(strings.ToLower(v), strings.ToLower(prefix)) {
			return v[len(prefix):], k, true
		}
	}
	return "", 0, false
}
