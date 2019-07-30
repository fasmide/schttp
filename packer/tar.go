package packer

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"
)

type TarGz struct {
	*gzip.Writer
	tar  *tar.Writer
	Path string
}

func NewTarGz(w io.Writer) *TarGz {
	gzip := gzip.NewWriter(w)
	return &TarGz{tar: tar.NewWriter(gzip), Writer: gzip}
}

func (z *TarGz) File(name string, mode os.FileMode, size int64, r io.Reader) error {
	err := z.tar.WriteHeader(&tar.Header{
		Name: path.Join(z.Path, name),

		// We remove 5 seconds as the tar "file is in the future" is highly annoying
		ModTime: time.Now().Add(time.Second * -5),
		Mode:    int64(mode),
		Size:    size,
	})
	if err != nil {
		return fmt.Errorf("unable to write tar header: %s, %d bytes: %s", name, size, err)
	}

	_, err = io.Copy(z.tar, r)
	if err != nil {
		return fmt.Errorf("unable to copy file contents: %s", err)
	}

	return nil
}

func (z *TarGz) Enter(name string, mode os.FileMode) error {
	z.Path = path.Join(z.Path, name)
	z.Path = path.Clean(z.Path)
	return nil
}

func (z *TarGz) Exit() error {
	parts := strings.Split(z.Path, "/")

	// if there was no path to split and we somehow received a directory leave
	if len(parts) == 0 {
		z.Path = "."
		return nil
	}

	z.Path = path.Join(parts[0 : len(parts)-1]...)
	return nil
}

func (z *TarGz) Close() error {
	err := z.tar.Close()
	if err != nil {
		// ninja also close the gzip
		z.Writer.Close()

		return fmt.Errorf("could not close tar: %s", err)
	}

	// also Flush and close gzip
	err = z.Writer.Close()
	if err != nil {
		return fmt.Errorf("could not close gzip: %s", err)
	}

	return nil
}
