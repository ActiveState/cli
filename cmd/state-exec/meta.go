package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/cmd/state-exec/internal/execmeta"
)

const (
	pathEnvVarKey     = "PATH"
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

func tranformedEnv(current []string, updates map[string]string) []string {
	env := envSliceToMap(os.Environ())
	for k, v := range updates {
		if k == pathEnvVarKey {
			p, ok := env[k]
			if ok {
				v += pathListSeparator + p
			}
			env[k] = v
		}
	}
	return envMapToSlice(env)
}

func envSliceToMap(envSlice []string) map[string]string {
	env := map[string]string{}
	for _, v := range envSlice {
		kv := strings.SplitN(v, "=", 2)
		env[kv[0]] = ""
		if len(kv) == 2 { // account for empty values, windows does some weird stuff, better safe than sorry
			env[kv[0]] = kv[1]
		}
	}
	return env
}

func envMapToSlice(envMap map[string]string) []string {
	var env []string
	for k, v := range envMap {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return env
}
