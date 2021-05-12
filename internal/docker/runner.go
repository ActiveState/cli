package docker

import (
	"io"
	"path/filepath"

	"github.com/ActiveState/cli/internal/constants"
	"github.com/ActiveState/cli/internal/errs"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/ActiveState/cli/internal/logging"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"golang.org/x/net/context"
)

type Runner struct {
	opts   TargetOptions
	client *client.Client
}

func NewRunner(opts TargetOptions) (*Runner, error) {
	client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, locale.WrapError(err, "err_docker_client", "Could not initialize Docker client, error received: {{.V0}}.", err.Error())
	}
	return &Runner{opts, client}, nil
}

func (r *Runner) Run(command []string, logWriter io.Writer) (runError error) {
	img := imageByCommit(r.client, r.opts.Commit)
	if img == nil {
		return locale.NewError("err_docker_run_img", "Could not find intermediary Docker image for your project.")
	}
	containerCfg := &container.Config{
		Env:   []string{constants.DockerLabelCommitEnvVarName + "=" + r.opts.Commit},
		Image: img.ID,
		Cmd:   command,
	}
	workdir := workDir(r.opts.YamlPath)
	hostCfg := &container.HostConfig{
		Mounts: []mount.Mount{{
			Type:   mount.TypeBind,
			Source: filepath.Dir(r.opts.YamlPath),
			Target: workdir,
			/*BindOptions: &mount.BindOptions{
			Propagation:  mount.PropagationShared,
			NonRecursive: false,
			},*/
		}},
	}
	container, err := r.client.ContainerCreate(context.Background(), containerCfg, hostCfg, nil, nil, "")
	if err != nil {
		return locale.WrapError(
			err, "err_docker_containercreate",
			"Could not create Docker container, returned error: {{.V0}}.",
			errs.Join(err, ": ").Error())
	}
	defer func() {
		if err := r.client.ContainerStop(context.Background(), container.ID, nil); err != nil {
			if runError == nil {
				runError = err
			} else {
				logging.Error("Could not stop container: %v", errs.Join(err, ": "))
			}
		}
		if err := r.client.ContainerRemove(context.Background(), container.ID, types.ContainerRemoveOptions{}); err != nil {
			if runError == nil {
				runError = err
			} else {
				logging.Error("Could not clean up container: %v", errs.Join(err, ": "))
			}
		}
	}()

	if len(container.Warnings) > 0 {
		for _, warning := range container.Warnings {
			logWriter.Write([]byte(warning + "\n"))
		}
	}

	if err := r.client.ContainerStart(context.Background(), container.ID, types.ContainerStartOptions{}); err != nil {
		return locale.WrapError(err, "err_docker_container_start",
			"Could not start container, returned error: {{.V0}}.", err.Error())
	}

	exec, err := r.client.ContainerExecCreate(context.Background(), container.ID, types.ExecConfig{
		Env:          []string{constants.DockerLabelCommitEnvVarName + "=" + r.opts.Commit},
		Cmd:          command,
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  true,
	})
	if err != nil {
		return locale.WrapError(err, "err_docker_container_exec",
			"Could not prepare executor, returned error: {{.V0}}.", err.Error())
	}

	attach, err := r.client.ContainerExecAttach(context.Background(), exec.ID, types.ExecStartCheck{})
	if err != nil {
		return locale.WrapError(err, "err_docker_container_execattach",
			"Could not attach to executor, returned error: {{.V0}}.", err.Error())
	}

	if _, err := io.Copy(logWriter, attach.Reader); err != nil {
		return locale.WrapError(err, "err_docker_container_iocopy",
			"Could not copy output stream, returned error: {{.V0}}.", err.Error())
	}

	return nil
}
