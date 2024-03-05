package sekaid

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/KiraCore/ryokai/internal/core/orchestration/docker"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/google/shlex"
)

type SekaiPlugin struct {
	dockerOrchestrator *docker.DockerOrchestrator
	sekaidConfig       *SekaidConfig
}

func NewSekaiPlugin(ctx context.Context) (*SekaiPlugin, error) {
	dockerOrchestrator, err := docker.NewDockerOrchestrator()
	if err != nil {
		return nil, err
	}

	return &SekaiPlugin{dockerOrchestrator: dockerOrchestrator}, nil
}

func (sekaiPlugin *SekaiPlugin) RunSekaidImageCommand(ctx context.Context, cmd string) error {
	hostFolderPath := filepath.Join(os.Getenv("HOME"), "real-folder-path")
	containerMountPath := "/volumes"

	// Ensure the host folder exists
	const dirPermissions = 0o755
	if err := os.MkdirAll(hostFolderPath, dirPermissions); err != nil {
		panic(fmt.Errorf("error when creating host folder: %w", err))
	}

	command, err := shlex.Split(cmd)
	if err != nil {
		return err
	}

	natRPCPort, err := nat.NewPort("tcp", sekaiPlugin.sekaidConfig.RpcPort)
	if err != nil {
		return err
	}

	natP2PPort, err := nat.NewPort("tcp", sekaiPlugin.sekaidConfig.P2PPort)
	if err != nil {
		return err
	}

	natPrometheusPort, err := nat.NewPort("tcp", sekaiPlugin.sekaidConfig.PrometheusPort)
	if err != nil {
		return err
	}

	containerConfig := &container.Config{
		Image: "ghcr.io/mrlutik/sekin:sekai_v0.3.41",
		Cmd:   command,
		Tty:   true,
		ExposedPorts: nat.PortSet{
			natRPCPort:        struct{}{},
			natP2PPort:        struct{}{},
			natPrometheusPort: struct{}{},
		},
	}

	hostConfig := &container.HostConfig{
		AutoRemove: true,
		Binds:      []string{hostFolderPath + ":" + containerMountPath},
		PortBindings: nat.PortMap{
			natRPCPort:        []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: sekaiPlugin.sekaidConfig.RpcPort}},
			natP2PPort:        []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: sekaiPlugin.sekaidConfig.P2PPort}},
			natPrometheusPort: []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: sekaiPlugin.sekaidConfig.PrometheusPort}},
		},
	}

	resp, err := sekaiPlugin.dockerOrchestrator.Cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		return err
	}

	if err := sekaiPlugin.dockerOrchestrator.Cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return err
	}

	return nil
}
