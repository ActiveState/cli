package docker

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/fileutils"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
)

type Builder struct {
	opts   TargetOptions
	client *client.Client
}

func NewBuilder(opts TargetOptions) (*Builder, error) {
	client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, locale.WrapError(err, "err_docker_client", "Could not initialize Docker client, error received: {{.V0}}.", err.Error())
	}
	return NewBuilderWithClient(opts, client)
}

func NewBuilderWithClient(opts TargetOptions, dockerClient *client.Client) (*Builder, error) {
	if !fileutils.TargetExists(opts.YamlPath) {
		return nil, errs.New("yaml does not exist: %s", opts.YamlPath)
	}
	builder := &Builder{opts, dockerClient}
	return builder, nil
}

func (b *Builder) Build(logWriter io.Writer) error {
	if b.TargetImageExists() {
		return nil
	}

	dir, err := b.prepareDockerDir()
	if err != nil {
		return locale.NewError("err_docker_build_prepare", "Could not create temporary Docker directory")
	}

	tar, err := archive.TarWithOptions(dir, &archive.TarOptions{})
	if err != nil {
		return errs.Wrap(err, "Could not create tar archive")
	}

	res, err := b.client.ImageBuild(context.Background(), tar, types.ImageBuildOptions{
		Labels: map[string]string{labelCommitKey: b.opts.Commit},
	})
	if res.Body != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return locale.NewError("err_docker_build_targetimage", "Could not build image, error returned: {{.V0}}.", errs.Join(err, ": ").Error())
	}

	if _, err := io.Copy(logWriter, res.Body); err != nil {
		return errs.Wrap(err, "Could not write logs")
	}

	return nil
}

func (b *Builder) TargetImageExists() bool {
	return imageByCommit(b.client, b.opts.Commit) != nil
}

func (b *Builder) prepareDockerDir() (string, error) {
	workdir := workDir(b.opts.YamlPath)

	dockerFile := fmt.Sprintf(
		`FROM %s
		ADD https://platform.activestate.com/dl/cli/install.sh /tmp/install.sh
		RUN TERM=xterm sh /tmp/install.sh -n
		WORKDIR %s
		COPY activestate.yaml ./activestate.yaml
		RUN VERBOSE=true state activate --mono --non-interactive
		CMD /bin/bash`,
		b.opts.Image, workdir)

	tempdir, err := ioutil.TempDir("", "state-script-docker")
	if err != nil {
		return "", errs.Wrap(err, "Could not create temp dir")
	}

	if err := fileutils.WriteFile(filepath.Join(tempdir, "Dockerfile"), []byte(dockerFile)); err != nil {
		return "", errs.Wrap(err, "Could not create Dockerfile")
	}

	if err := fileutils.CopyFile(b.opts.YamlPath, filepath.Join(tempdir, filepath.Base(b.opts.YamlPath))); err != nil {
		return "", errs.Wrap(err, "Could not copy activestate.yaml to temp dir")
	}

	return tempdir, nil
}
