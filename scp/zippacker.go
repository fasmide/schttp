package scp

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"
)

type ZipPacker struct {
	*zip.Writer
	Path string
}

func NewZipPacker(w io.Writer) *ZipPacker {
	return &ZipPacker{Writer: zip.NewWriter(w)}
}

func (z *ZipPacker) File(name string, mode os.FileMode, r io.Reader) error {
	fd, err := z.CreateHeader(&zip.FileHeader{
		Name:     path.Join(z.Path, name),
		Modified: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("unable to create file: %s", err)
	}

	_, err = io.Copy(fd, r)
	if err != nil {
		return fmt.Errorf("unable to copy file contents: %s", err)
	}

	return nil
}

func (z *ZipPacker) Enter(name string, mode os.FileMode) error {
	z.Path = path.Join(z.Path, name)
	z.Path = path.Clean(z.Path)
	return nil
}

func (z *ZipPacker) Exit() error {
	parts := strings.Split(z.Path, "/")

	// if there was no path to split and we somehow received a directory leave
	if len(parts) == 0 {
		z.Path = "."
		return nil
	}

	z.Path = path.Join(parts[0 : len(parts)-1]...)
	return nil
}
