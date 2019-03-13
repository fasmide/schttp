package scp

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

type TarPacker struct {
	*gzip.Writer
	tar  *tar.Writer
	Path string
}

func NewTarPacker(w io.Writer) *TarPacker {
	gzip := gzip.NewWriter(w)
	return &TarPacker{tar: tar.NewWriter(gzip)}
}

func (z *TarPacker) File(name string, mode os.FileMode, r io.Reader) error {
	err := z.tar.WriteHeader(&tar.Header{
		Name:    path.Join(z.Path, name),
		ModTime: time.Now(),
		Mode:    int64(mode),
	})
	if err != nil {
		return fmt.Errorf("unable to create file: %s", err)
	}

	_, err = io.Copy(z.tar, r)
	if err != nil {
		return fmt.Errorf("unable to copy file contents: %s", err)
	}

	return nil
}

func (z *TarPacker) Enter(name string, mode os.FileMode) error {
	z.Path = path.Join(z.Path, name)
	z.Path = path.Clean(z.Path)
	return nil
}

func (z *TarPacker) Exit() error {
	parts := strings.Split(z.Path, "/")

	// if there was no path to split and we somehow received a directory leave
	if len(parts) == 0 {
		z.Path = "."
		return nil
	}

	z.Path = path.Join(parts[0 : len(parts)-1]...)
	return nil
}

func (z *TarPacker) Close() error {
	err := z.tar.Close()
	if err != nil {
		// ninja also close the gzip
		z.Writer.Close()

		return fmt.Errorf("could not close tar: %s", err)
	}

	// also close gzip
	err = z.Writer.Close()
	if err != nil {
		return fmt.Errorf("could not close gzip: %s", err)
	}

	return nil
}
