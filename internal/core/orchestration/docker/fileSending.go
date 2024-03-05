package docker

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types"
)

var (
	ErrPackageInstallationFailed = errors.New("package installation failed")
	ErrFileNotFoundInTarBase     = errors.New("file not found in tar archive")
	ErrStderrNotEmpty            = errors.New("stderr is not empty")
)

// SendFileToContainer sends a file from the host machine to a specified directory inside a Docker container.
// - ctx: The context for the operation.
// - filePathOnHostMachine: The path of the file on the host machine.
// - directoryPathOnContainer: The path of the directory inside the container where the file will be copied.
// - containerID: The ID or name of the Docker container.
// Returns an error if any issue occurs during the file sending process.
func (dm *DockerOrchestrator) SendFileToContainer(ctx context.Context, filePathOnHostMachine, directoryPathOnContainer, containerID string) error { //nolint:lll,funlen
	file, err := os.Open(filePathOnHostMachine)
	if err != nil {
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	tarWriter := tar.NewWriter(&buf)

	err = addFileToTar(fileInfo, file, tarWriter)
	if err != nil {
		return err
	}

	err = tarWriter.Close()
	if err != nil {
		return err
	}

	tarContent := buf.Bytes()
	tarReader := bytes.NewReader(tarContent)
	copyOptions := types.CopyToContainerOptions{ //nolint:exhaustruct
		AllowOverwriteDirWithFile: false,
	}

	err = dm.Cli.CopyToContainer(ctx, containerID, directoryPathOnContainer, tarReader, copyOptions)
	if err != nil {
		return err
	}

	return nil
}

func addFileToTar(fileInfo os.FileInfo, file io.Reader, tarWriter *tar.Writer) error {
	header := &tar.Header{
		Name: fileInfo.Name(),
		Mode: int64(fileInfo.Mode()),
		Size: fileInfo.Size(),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	if _, err := io.Copy(tarWriter, file); err != nil {
		return err
	}

	return nil
}

func (dm *DockerOrchestrator) WriteFileDataToContainer(ctx context.Context, fileData []byte, fileName, destPath, containerID string) error { //nolint:lll
	tarBuffer := new(bytes.Buffer)
	tarWriter := tar.NewWriter(tarBuffer)

	header := &tar.Header{
		Name: fileName,
		Mode: 0o644,
		Size: int64(len(fileData)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return err
	}

	if _, err := tarWriter.Write(fileData); err != nil {
		return err
	}

	if err := tarWriter.Close(); err != nil {
		return err
	}

	err := dm.Cli.CopyToContainer(ctx, containerID, destPath, tarBuffer, types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	})
	if err != nil {
		return err
	}

	return nil
}

// GetFileFromContainer retrieves a file from a specified container using the Docker API.
// It copies the TAR archive with file from the specified folder path in the container,
// read file from TAR archive and returns the file content as a byte slice.
func (dm *DockerOrchestrator) GetFileFromContainer(ctx context.Context, folderPathOnContainer, fileName, containerID string) ([]byte, error) { //nolint:lll
	readCloser, _, err := dm.Cli.CopyFromContainer(ctx, containerID, folderPathOnContainer+"/"+fileName)
	if err != nil {
		return nil, (fmt.Errorf("error when copying from container, error: %w", err))
	}
	defer readCloser.Close()

	tr := tar.NewReader(readCloser)

	outBytes, err := readTarArchive(tr, fileName)
	if err != nil {
		return nil, err
	}

	return outBytes, nil
}

// Todo: this func has to be deprecated, use volume folder instead
// readTarArchive reads a file from the TAR archive stream
// and returns the file content as a byte slice.
func readTarArchive(tarReader *tar.Reader, fileName string) ([]byte, error) {
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("error when advancing to next tar archive entry: %w", err)
		}

		if hdr.Name == fileName {
			b, err := io.ReadAll(tarReader) //nolint: govet
			if err != nil {
				return nil, fmt.Errorf("error when reading tar file, error: %w", err)
			}

			return b, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrFileNotFoundInTarBase, fileName)
}
