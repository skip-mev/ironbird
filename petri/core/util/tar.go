package util

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
)

// MakeSingleFileTar creates a tar archive of a single file.
// It returns the tar archive as an io.Reader object
func MakeSingleFileTar(name string, file io.Reader) (io.Reader, error) {
	b := bytes.NewBuffer([]byte{})

	tw := tar.NewWriter(b)

	var fileSize int64

	if r, ok := file.(interface{ Size() int64 }); ok {
		fileSize = r.Size()
	} else {
		return nil, errors.New("unable to determine file size")
	}

	header := &tar.Header{
		Name: name,
		Size: fileSize,
		Mode: 0o777,
	}

	if err := tw.WriteHeader(header); err != nil {
		return nil, err
	}

	_, err := io.Copy(tw, file)
	if err != nil {
		return nil, err
	}

	return b, nil
}
