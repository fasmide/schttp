package scp

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"
)

type ZipPacker struct {
	*zip.Writer
	Path string
}

func NewZipPacker(w io.Writer) *ZipPacker {
	return &ZipPacker{Writer: zip.NewWriter(w)}
}

func (z *ZipPacker) File(name string, mode os.FileMode, r io.Reader) error {
	fd, err := z.Create(path.Join(z.Path, name))
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
	log.Printf("ZipPacker: Creating directory %s with mode %s", name, mode)
	return nil
}

func (z *ZipPacker) Exit() error {

	log.Printf("ZipPacker: Leaving %s", z.Path)
	parts := strings.Split(z.Path, "/")
	z.Path = path.Join(parts[0 : len(parts)-1]...)
	return nil
}
