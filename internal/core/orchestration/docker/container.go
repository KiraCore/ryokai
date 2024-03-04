package docker

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	ryokaiTypes "github.com/KiraCore/ryokai/pkg/ryokaicommon/types"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/google/shlex"
)

// ExecCommandInContainer executes a command inside a specified container.
// ctx: The context for the operation.
// containerID: The ID or name of the container.
// command: The command to execute inside the container.
// Returns the output of the command as a byte slice and an error if any issue occurs during the command execution.
func (dm *DockerOrchestrator) ExecCommandInContainer(ctx context.Context, containerID, command string) ([]byte, error) { //nolint funlen
	cmdArray, err := shlex.Split(command)
	if err != nil {
		slog.Error("Error when splitting command string", "error", err)

		return nil, err
	}

	slog.Info("Running command ", "command", command, "containerID", containerID)

	execCreateResponse, err := dm.Cli.ContainerExecCreate(ctx, containerID, types.ExecConfig{ //nolint:exhaustruct
		Cmd:          cmdArray,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		slog.Error("Exec configuration error: %s", err)

		return nil, err
	}

	resp, err := dm.Cli.ContainerExecAttach(ctx, execCreateResponse.ID, types.ExecStartCheck{})
	if err != nil {
		slog.Error("Error when executing command", "command", command, "error", err)

		return nil, err
	}
	defer resp.Close()

	var outBuf, errBuf bytes.Buffer

	_, err = stdcopy.StdCopy(&outBuf, &errBuf, resp.Reader)
	if err != nil {
		slog.Error("Reading response error", "err", err)

		return errBuf.Bytes(), err
	}

	slog.Info("Running successfully", "command", command)

	return outBuf.Bytes(), nil
}

func (dm *DockerOrchestrator) CreateVolume(ctx context.Context, volumeCreateOption volume.CreateOptions) (volume.Volume, error) { //nolint:lll
	vol, err := dm.Cli.VolumeCreate(ctx, volumeCreateOption)
	if err != nil {
		return volume.Volume{}, fmt.Errorf("error when creating volume %w", err)
	}

	return vol, nil
}

func (dm *DockerOrchestrator) CreateContainer(ctx context.Context, spec ryokaiTypes.ContainerSpec) (string, error) {
	slog.Info("Creating")

	resp, err := dm.Cli.ContainerCreate(ctx, &container.Config{
		Image: spec.Image,
		Env:   spec.Env,
	}, nil, nil, nil, "")
	if err != nil {
		return "", err
	}

	return resp.ID, nil
}

func (dm *DockerOrchestrator) StartContainer(ctx context.Context, containerID string) error {
	if err := dm.Cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil { //nolint exhaustruct
		return err
	}

	return nil
}

func (dm *DockerOrchestrator) StopContainer(ctx context.Context, containerID string) error {
	return nil
}

func (dm *DockerOrchestrator) RemoveContainer(ctx context.Context, containerID string) error {
	return nil
}
