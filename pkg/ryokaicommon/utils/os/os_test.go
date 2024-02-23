package os_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	osUtils "github.com/KiraCore/ryokai/pkg/ryokaicommon/utils/os"
)

func TestCopyFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Error("Unable to create directory in temporary folder")
	}
	srcTempFile := tempDir + "/srcFile"
	dstTempFile := tempDir + "/dstFile"
	nonExistingPath := "/pathThatShould/not/exist/TestCopyFile"
	dataToWrite := []byte("TestString")

	file, err := os.Create(srcTempFile)
	if err != nil {
		t.Error("Unable to create temporary file")
		t.FailNow()
	}
	_, err = file.Write(dataToWrite)
	if err != nil {
		t.Error("Unable to write data to temporary file")
		t.FailNow()
	}
	file.Close()

	tmpData, err := os.ReadFile(srcTempFile)
	if err != nil {
		t.Error("Unable to read temporary file")
		t.FailNow()
	}
	if string(tmpData) != string(dataToWrite) {
		t.Error("Data that has been written is wrong")
		t.FailNow()
	}

	tests := []struct {
		name        string
		srcPath     string
		dstPath     string
		dataToWrite []byte
		wantErr     bool
	}{
		{
			name:        "normal behavior",
			srcPath:     srcTempFile,
			dstPath:     dstTempFile,
			dataToWrite: tmpData,
			wantErr:     false,
		},
		{
			name:        "source doesn't exist",
			srcPath:     nonExistingPath,
			dstPath:     dstTempFile,
			dataToWrite: tmpData,
			wantErr:     true,
		},
		{
			name:        "destination doesn't exist",
			srcPath:     srcTempFile,
			dstPath:     nonExistingPath,
			dataToWrite: tmpData,
			wantErr:     true,
		},
		{
			name:        "same path for source and destination",
			srcPath:     srcTempFile,
			dstPath:     srcTempFile,
			dataToWrite: tmpData,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		err := osUtils.CopyFile(tt.srcPath, tt.dstPath)
		if (err != nil) != tt.wantErr {
			t.Errorf("CheckIfPathExist(%s, %s) err = %v want %v\n", tt.srcPath, tt.dstPath, err, tt.wantErr)
		}
		if err == nil {
			tmpData, err := os.ReadFile(tt.dstPath)
			if err != nil {
				t.Errorf("Unable to create temporary file, err: %v", err)
			} else {
				if string(tmpData) != string(tt.dataToWrite) {
					t.Error("Data that has been written is wrong")
				}
			}

		}
	}
	err = os.RemoveAll(tempDir)
	if err != nil {
		t.Error("Unable to delete directory in temporary folder")
	}
}

func TestCreateFileWithData(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Error("Unable to create directory in temporary folder")
	}
	tests := []struct {
		name        string
		path        string
		dataToWrite []byte
		wantErr     bool
	}{
		{
			name:        "Check normal behavior",
			path:        tempDir + "/TestCreateFileWithData",
			dataToWrite: []byte("DataToWrite"),
			wantErr:     false,
		},
		{
			name:        "Check if data is empty",
			path:        tempDir + "/TestCreateEmptyFile",
			dataToWrite: []byte{},
			wantErr:     false,
		},
		{
			name:        "Check if path doesn't exist",
			path:        "/pathThatShould/not/exist/for/file",
			dataToWrite: []byte("DataToWrite"),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		err = osUtils.CreateFileWithData(tt.path, tt.dataToWrite)
		if (err != nil) != tt.wantErr {
			t.Errorf("CreateFileWithData(%s) err = %v want %v\n", tt.path, err, tt.wantErr)
		}
		if err == nil {
			tmpData, err := os.ReadFile(tt.path)
			if err != nil {
				t.Error("Unable to read temporary file")
			}
			if string(tt.dataToWrite) != string(tmpData) {
				t.Error("Files are not equal")
			}

		}
	}
	err = os.RemoveAll(tempDir)
	if err != nil {
		t.Error("Unable to delete directory in temporary folder")
	}
}

func TestCheckIfPortIsValid(t *testing.T) {
	const portValidRange = 65535
	validPorts := []string{"0", "3", "55", "22222", strconv.Itoa(portValidRange)}
	// checking valid port range
	for _, num := range validPorts {
		ok := osUtils.CheckIfPortIsValid(num)
		if !ok {
			t.Errorf("Error checking <%s> is in valid range", num)
		}
	}
	// checking negative port range
	invalidPorts := []string{"-1", "-5", "-2000", "-50434", "-portValidRange", "number", "334combined"}
	for _, num := range invalidPorts {
		ok := osUtils.CheckIfPortIsValid(num)
		if ok {
			t.Errorf("Error checking <%v> port. Port is in invalid range", num)
		}
	}
}

func TestRunCommand(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Error("Unable to create directory in temporary folder")
		t.FailNow()
	}

	tests := []struct {
		name    string
		cmd     string
		wantErr bool
	}{
		{
			name:    "normal execution",
			cmd:     "ls",
			wantErr: false,
		},
		{
			name:    "normal execution with args",
			cmd:     fmt.Sprintf("ls %s", tempDir),
			wantErr: false,
		},
		{
			name:    "error execution",
			cmd:     fmt.Sprintf("lsssss %s", tempDir),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		_, err = osUtils.RunCommand(tt.cmd)
		if (err != nil) != tt.wantErr {
			t.Errorf("RunCommand(%s) err = %v want %v\n", tt.cmd, err, tt.wantErr)
		}
	}

	err = os.RemoveAll(tempDir)
	if err != nil {
		t.Error("Unable to delete directory in temporary folder")
	}
}
