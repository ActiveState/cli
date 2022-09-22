package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ActiveState/cli/internal/osutils"
	"github.com/ActiveState/cli/pkg/platform/runtime/executor"
)

const (
	pathEnvVarKey     = "PATH"
	pathListSeparator = string(os.PathListSeparator)
)

type executorMeta struct {
	*executor.Meta
	MatchingBin    string
	TransformedEnv []string
}

func newExecutorMeta(execPath string) (*executorMeta, error) {
	execDir := filepath.Dir(execPath)
	metaPath := filepath.Join(execDir, executor.MetaFileName)
	meta, err := executor.NewMetaFromFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("create new executor meta: %w", err)
	}

	em := executorMeta{
		Meta:           meta,
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

func tranformedEnv(current []string, updates map[string]string) []string {
	env := osutils.EnvSliceToMap(os.Environ())
	for k, v := range updates {
		if k == pathEnvVarKey {
			p, ok := env[k]
			if ok {
				v += pathListSeparator + p
			}
			env[k] = v
		}
	}
	return osutils.EnvMapToSlice(env)
}
