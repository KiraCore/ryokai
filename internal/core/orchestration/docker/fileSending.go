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

// SendFileToContainer sends a file from the host machine to a specified directory inside a Docker container.
// - ctx: The context for the operation.
// - filePathOnHostMachine: The path of the file on the host machine.
// - directoryPathOnContainer: The path of the directory inside the container where the file will be copied.
// - containerID: The ID or name of the Docker container.
// Returns an error if any issue occurs during the file sending process.
func (dm *DockerOrchestrator) SendFileToContainer(ctx context.Context, filePathOnHostMachine, directoryPathOnContainer, containerID string) error {
	// log.Infof("Sending file '%s' to container '%s' to '%s'", filePathOnHostMachine, containerID, directoryPathOnContainer)
	file, err := os.Open(filePathOnHostMachine)
	if err != nil {
		// log.Errorf("Opening file error: %s", err)
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		// log.Errorf("Can't open file stat: %s", err)
		return err
	}

	var buf bytes.Buffer
	tarWriter := tar.NewWriter(&buf)

	err = addFileToTar(fileInfo, file, tarWriter)
	if err != nil {
		// log.Errorf("Adding file to tar error: %s", err)
		return err
	}

	err = tarWriter.Close()
	if err != nil {
		// log.Errorf("Closing tar error: %s", err)
		return err
	}

	tarContent := buf.Bytes()
	tarReader := bytes.NewReader(tarContent)
	copyOptions := types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: false,
	}

	err = dm.Cli.CopyToContainer(ctx, containerID, directoryPathOnContainer, tarReader, copyOptions)
	if err != nil {
		// log.Errorf("Copying tar to container error: %s", err)
		return err
	}

	// log.Infof("Successfully copied '%s' to '%s' in '%s' container", filePathOnHostMachine, directoryPathOnContainer, containerID)
	return nil
}

func addFileToTar(fileInfo os.FileInfo, file io.Reader, tarWriter *tar.Writer) error {
	// log.Infof("Writing file '%s' to tar archive", fileInfo.Name())

	header := &tar.Header{
		Name: fileInfo.Name(),
		Mode: int64(fileInfo.Mode()),
		Size: fileInfo.Size(),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		// log.Errorf("Writing tar header error: %s", err)
		return err
	}

	if _, err := io.Copy(tarWriter, file); err != nil {
		// log.Errorf("Copying error: %s", err)
		return err
	}

	return nil
}

func (dm *DockerOrchestrator) WriteFileDataToContainer(ctx context.Context, fileData []byte, fileName, destPath, containerID string) error {
	// log.Infof("Writing file to container '%s'", containerID)

	tarBuffer := new(bytes.Buffer)
	tw := tar.NewWriter(tarBuffer)

	header := &tar.Header{
		Name: fileName,
		Mode: 0o644,
		Size: int64(len(fileData)),
	}
	if err := tw.WriteHeader(header); err != nil {
		// log.Errorf("Writing tar header error: %s", err)
		return err
	}

	if _, err := tw.Write(fileData); err != nil {
		// log.Errorf("Writing file data to tar error: %s", err)
		return err
	}

	if err := tw.Close(); err != nil {
		// log.Errorf("Closing tar writer error: %s", err)
		return err
	}

	err := dm.Cli.CopyToContainer(ctx, containerID, destPath, tarBuffer, types.CopyToContainerOptions{
		AllowOverwriteDirWithFile: true,
	})
	if err != nil {
		// log.Errorf("Failed to copy file to container '%s': %s", containerID, err)
		return err
	}

	// log.Infof("File '%s' is successfully written on '%s' in container '%s'", fileName, destPath, containerID)

	return nil
}

// GetFileFromContainer retrieves a file from a specified container using the Docker API.
// It copies the TAR archive with file from the specified folder path in the container,
// read file from TAR archive and returns the file content as a byte slice.
func (dm *DockerOrchestrator) GetFileFromContainer(ctx context.Context, folderPathOnContainer, fileName, containerID string) ([]byte, error) {
	// log.Infof("Getting file '%s' from container '%s'", fileName, folderPathOnContainer)

	rc, _, err := dm.Cli.CopyFromContainer(ctx, containerID, folderPathOnContainer+"/"+fileName)
	if err != nil {
		// log.Errorf("Copying from container error: %s", err)
		return nil, err
	}
	defer rc.Close()

	tr := tar.NewReader(rc)
	b, err := readTarArchive(tr, fileName)
	if err != nil {
		// log.Errorf("Reading Tar archive error: %s", err)
		return nil, err
	}

	return b, nil
}

// readTarArchive reads a file from the TAR archive stream
// and returns the file content as a byte slice.
func readTarArchive(tr *tar.Reader, fileName string) ([]byte, error) {
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Name == fileName {
			b, err := io.ReadAll(tr)
			if err != nil {
				return nil, err
			}
			return b, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrFileNotFoundInTarBase, fileName)
}

var (
	ErrPackageInstallationFailed = errors.New("package installation failed")
	ErrFileNotFoundInTarBase     = errors.New("file not found in tar archive")
	ErrStderrNotEmpty            = errors.New("stderr is not empty")
)
