package docker

import (
	"path/filepath"
	"strings"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

const labelCommitKey = "activestate-commit"

type TargetOptions struct {
	Image    string
	Commit   string
	YamlPath string
}

func imageByCommit(client *client.Client, commit string) *types.ImageSummary {
	images, err := client.ImageList(context.Background(), types.ImageListOptions{All: true})
	if err != nil {
		logging.Error("Could not inspect docker image: %v", errs.Join(err, ": ").Error())
	}

	for _, img := range images {
		if v, ok := img.Labels[labelCommitKey]; ok && v == commit {
			return &img
		}
	}

	return nil
}

func workDir(yamlPath string) string {
	workdir := "/home/activestate"
	isPosixPath := strings.HasPrefix(yamlPath, "/")
	if isPosixPath {
		workdir = filepath.Dir(yamlPath)
	}
	return workdir
}
