package packer

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"
)

type Zip struct {
	*zip.Writer
	Path string
}

func NewZip(w io.Writer) *Zip {
	return &Zip{Writer: zip.NewWriter(w)}
}

func (z *Zip) File(name string, mode os.FileMode, _ int64, r io.Reader) error {
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

func (z *Zip) Enter(name string, mode os.FileMode) error {
	z.Path = path.Join(z.Path, name)
	z.Path = path.Clean(z.Path)
	return nil
}

func (z *Zip) Exit() error {
	parts := strings.Split(z.Path, "/")

	// if there was no path to split and we somehow received a directory leave
	if len(parts) == 0 {
		z.Path = "."
		return nil
	}

	z.Path = path.Join(parts[0 : len(parts)-1]...)
	return nil
}
