package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
)

func (dm *DockerOrchestrator) PullImage(ctx context.Context, imageName string) error {
	options := types.ImagePullOptions{}
	reader, err := dm.Cli.ImagePull(ctx, imageName, options)
	if err != nil {
		// log.Errorf("failed to pull image: %s", err)
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()

	// Create a buffer for the reader
	buf := new(bytes.Buffer)

	// Copy the image pull output to the buffer
	_, err = io.Copy(buf, reader)
	if err != nil {
		// log.Errorf("failed to copy image pull output: %s", err)
		return fmt.Errorf("failed to copy image pull output: %w", err)
	}

	// Print the prettified output from the buffer
	// log.Infof("Image pull output: %s", buf.String())

	return nil
}
