package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
)

func (dm *DockerOrchestrator) PullImage(ctx context.Context, image string) error {
	reader, err := dm.Cli.ImagePull(ctx, image, types.ImagePullOptions{}) //nolint:exhaustruct
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()

	// Create a buffer for the reader
	buf := new(bytes.Buffer)

	// Copy the image pull output to the buffer
	_, err = io.Copy(buf, reader)
	if err != nil {
		return fmt.Errorf("failed to copy image pull output: %w", err)
	}

	// Print the prettified output from the buffer
	// log.Infof("Image pull output: %s", buf.String())

	return nil
}
