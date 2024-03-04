package docker

import (
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type DockerOrchestrator struct {
	containerConfig container.Config
	hostConfigg     container.HostConfig
	Cli             *client.Client
}

func NewDockerOrchestrator() (*DockerOrchestrator, error) {
	cli, err := client.NewClientWithOpts(client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &DockerOrchestrator{Cli: cli}, nil
}
