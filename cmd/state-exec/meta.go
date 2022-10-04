package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/pkg/platform/runtime/executor/execmeta"
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
		return nil, fmt.Errorf("create new executor meta: %w", err)
	}

	em := executorMeta{
		ExecMeta:       meta,
		MatchingBin:    matchingBinByPath(meta.Bins, execPath),
		TransformedEnv: tranformedEnv(os.Environ(), meta.Env),
	}

	return &em, nil
}

func matchingBinByPath(bins []string, path string) string {
	name := filepath.Base(path)
	for _, bin := range bins {
		if filepath.Base(bin) == name {
			return bin
		}
	}
	return ""
}

func tranformedEnv(current []string, updates []string) []string {
	for _, update := range updates {
		if strings.HasPrefix(update, pathEnvVarPrefix) {
			pathUpdate := update[len(pathEnvVarPrefix):]
			if pathCurrent, ok := getEnvVarValue(current, pathEnvVarPrefix); ok {
				pathUpdate += pathListSeparator + pathCurrent
			}
			update = pathEnvVarPrefix + pathUpdate
		}
		current = append(current, update)
	}
	return current
}

// getEnvVarValue returns the value of the environment variable from the
// provided environment slice. The prefix should end with the environment
// variable separator.
func getEnvVarValue(env []string, prefix string) (string, bool) {
	for _, v := range env {
		if strings.HasPrefix(v, prefix) {
			return v[len(prefix):], true
		}
	}
	return "", false
}
